package connection

import (
	"bytes"
	"fmt"
	"github.com/ztrue/tracerr"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

type ConnectType string
type ConnectionStatus string

const SSH ConnectType = "SSH"
const Telnet ConnectType = "Telnet"

const ConnectionOpened ConnectionStatus = "opened"
const ConnectionClosed ConnectionStatus = "closed"
const ConnectionLogined ConnectionStatus = "logined"
const ConnectionErrorLogon ConnectionStatus = "error_logon"
const ConnectionBinded ConnectionStatus = "binded"

type Connection struct {
	Ip                  string      `json:"ip"`
	Port                int         `json:"port"`
	Type                ConnectType `json:"type"`
	conn                net.Conn
	Conf                Config `json:"conf"`
	prompt              *Prompt
	Labels              map[string]interface{} `json:"labels"`
	globalBuffer        string
	Status              ConnectionStatus `json:"status"`
	LastInteractiveTime time.Time        `json:"last_interactive_time"`
}

func Init(Ip string, Port int, Type ConnectType) *Connection {
	conn := new(Connection)
	conn.Labels = make(map[string]interface{})
	return conn
}

func (c *Connection) OpenConnection() error {

	if c.Type == Telnet {
		lg.InfoF("Try open connection to %v:%v over telnet", c.Ip, c.Port)
		if err := c.openTelnetConnection(); err != nil {
			c.Status = ConnectionClosed
			lg.Errorf("Error open connection to %v:%v - %v", c.Ip, c.Port, err.Error())
			return err
		}
		lg.InfoF("Connection to %v:%v over telnet opened", c.Ip, c.Port)

	}
	if c.Type == SSH {
		lg.Errorf("SSH connections not supported at this time!!!")
		return fmt.Errorf("SSH connections not supported at this time!!!")
	}
	c.Status = ConnectionOpened
	return nil
}

func (c *Connection) openTelnetConnection() error {
	d := net.Dialer{
		Timeout: c.Conf.ConnTimeout,
	}
	telnet, err := d.Dial("tcp", fmt.Sprintf("%v:%v", c.Ip, c.Port))
	c.conn = telnet
	return err
}

func (c *Connection) Login(login, password string) error {
	if c.Type == Telnet {
		if err, _ := c.Wait(c.prompt.Login, nil); err != nil {
			lg.Errorf("Error wait prompt for %v:%v - %v", c.Ip, c.Port, err.Error())
			c.Status = ConnectionErrorLogon
			return tracerr.Wrap(fmt.Errorf("error wait prompt '%v' in login for %v:%v - %v", c.prompt.Login, c.Ip, c.Port, tracerr.Sprint(err)))
		}
		if err, _ := c.Command(login, c.prompt.Password, true, nil); err != nil {
			lg.Errorf("Error wait prompt for %v:%v - %v", c.Ip, c.Port, err.Error())
			c.Status = ConnectionErrorLogon
			return tracerr.Wrap(fmt.Errorf("error wait prompt '%v' in wait password for %v:%v - %v", c.prompt.Password, c.Ip, c.Port, tracerr.Sprint(err)))
		}
		if err, _ := c.Command(password, c.prompt.Command, true, nil); err != nil {
			lg.Errorf("Error wait prompt for %v:%v - %v", c.Ip, c.Port, err.Error())
			c.Status = ConnectionErrorLogon
			return tracerr.Wrap(fmt.Errorf("error wait prompt '%v' in wait password for %v:%v - %v", c.prompt.Command, c.Ip, c.Port, tracerr.Sprint(err)))
		}
	} else if c.Type == SSH {
		if err, _ := c.Wait(c.prompt.Password, nil); err != nil {
			lg.Errorf("Error wait prompt for %v:%v - %v", c.Ip, c.Port, err.Error())
			c.Status = ConnectionErrorLogon
			return tracerr.Wrap(fmt.Errorf("error wait prompt '%v' in login for %v:%v - %v", c.prompt.Login, c.Ip, c.Port, tracerr.Sprint(err)))
		}
		if err, _ := c.Command(password, c.prompt.Command, true, nil); err != nil {
			lg.Errorf("Error wait prompt for %v:%v - %v", c.Ip, c.Port, err.Error())
			c.Status = ConnectionErrorLogon
			return tracerr.Wrap(fmt.Errorf("error wait prompt '%v' in wait password for %v:%v - %v", c.prompt.Command, c.Ip, c.Port, tracerr.Sprint(err)))
		}
	}
	c.Status = ConnectionLogined
	return nil
}

func (c *Connection) Wait(prompt string, wait *time.Duration) (error, string) {
	b := make([]byte, 1)
	var buf bytes.Buffer
	match, err := regexp.Compile(prompt)
	if err != nil {
		lg.Errorf("Error compile regular prompt '%v' for device with IP %v", prompt, c.Ip)
		return tracerr.Wrap(err), ""
	}
	if wait == nil {
		wait = &(c.Conf.ConnTimeout)
	}
	for {
		_, err := c.conn.Read(b)
		if err == io.EOF {
			lg.Errorf("Receive EOF byte on device %v", c.Ip)
			return tracerr.Wrap(err), ""
		} else if err != nil {
			lg.Errorf("error read from client: %v", tracerr.Wrap(err))
			return tracerr.Wrap(err), ""
		}
		if err := c.conn.SetDeadline(time.Now().Add(*wait)); err != nil {
			lg.Errorf("Error set deadline timeout for device with IP %v", c.Ip)
			return tracerr.Wrap(err), ""
		}
		if bytes.Contains([]byte{0x00, 0xFF, 0x02, 0x01, 0x03, 0x04, 0x07, 0x08, byte(246)}, b) {
			continue
		}
		buf.Write(b)
		lines := buf.String()
		c.LastInteractiveTime = time.Now()

		for _, line := range strings.Split(buf.String(), "\n") {
			if match.MatchString(line) {
				c.globalBuffer = lines
				return nil, lines
			}
		}
	}
}

func (c *Connection) Command(command, prompt string, addNewLine bool, wait *time.Duration) (error, string) {
	if err := c.Write(command, addNewLine, wait); err != nil {
		return tracerr.Wrap(err), ""
	}
	return c.Wait(prompt, wait)
}

func (c *Connection) Write(command string, addNewLine bool, wait *time.Duration) error {
	if addNewLine {
		command += "\n"
	}
	c.globalBuffer += command
	if wait == nil {
		wait = &(c.Conf.ConnTimeout)
	}
	if err := c.conn.SetDeadline(time.Now().Add(*wait)); err != nil {
		lg.Errorf("Error set deadline timeout for device with IP %v", c.Ip)
		return tracerr.Wrap(err)
	}
	if _, err := c.conn.Write([]byte(command)); err != nil {
		lg.Errorf("Error write command '%v' to device with IP %v", command, c.Ip)
		return tracerr.Wrap(err)
	}
	c.LastInteractiveTime = time.Now()
	return nil
}

func (c *Connection) writeByte(bt byte, wait *time.Duration) error {
	if wait == nil {
		wait = &(c.Conf.ConnTimeout)
	}
	if err := c.conn.SetDeadline(time.Now().Add(*wait)); err != nil {
		lg.Errorf("Error set deadline timeout for device with IP %v", c.Ip)
		return tracerr.Wrap(err)
	}
	if _, err := c.conn.Write([]byte{bt}); err != nil {
		lg.Errorf("Error write command '%v' to device with IP %v", bt, c.Ip)
		return tracerr.Wrap(err)
	}
	c.LastInteractiveTime = time.Now()
	return nil
}

func (c *Connection) readByte() (error, byte) {
	b := make([]byte, 1)
	_, err := c.conn.Read(b)
	if err == io.EOF {
		lg.Errorf("Receive EOF byte on device %v", c.Ip)
		return tracerr.Wrap(err), b[0]
	} else if err != nil {
		lg.Errorf("error read from client: %v", tracerr.Wrap(err))
		return tracerr.Wrap(err), b[0]
	}

	c.LastInteractiveTime = time.Now()
	return nil, b[0]
}

func (c *Connection) GetGlobalBuffer() string {
	return c.globalBuffer
}

func (c *Connection) SendPing() error {
	if err := c.writeByte(byte(246), nil); err != nil {
		return tracerr.Wrap(err)
	}
	if err, _ := c.readByte(); err != nil {
		return tracerr.Wrap(err)
	}
	c.LastInteractiveTime = time.Now()
	return nil
}

func (c *Connection) CloseConnection() error {
	c.Status = ConnectionClosed
	if err := c.conn.Close(); err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

func (c *Connection) SendAfterLoginCommands() error {
	for _, command := range c.Conf.AfterLoginCommands {
		if err, _ := c.Command(command, c.prompt.Command, true, nil); err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (c *Connection) SendBeforeLogoutCommands(wait *time.Duration) error {
	for _, command := range c.Conf.AfterLoginCommands {
		if err, _ := c.Command(command, c.prompt.Command, true, wait); err != nil {
			return tracerr.Wrap(err)
		}
	}
	return nil
}

func (c *Connection) Bind() {

}
