package device

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

const SSH ConnectType = "SSH"
const Telnet ConnectType = "Telnet"

type Connection struct {
	Ip                 string
	Port               int
	Type               ConnectType
	conn               net.Conn
	prompt             *Prompt
	Labels             map[string]interface{}
	lastCommandRawData string
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
			lg.Errorf("Error open connection to %v:%v - %v", c.Ip, c.Port, err.Error())
			return err
		}
		lg.InfoF("Connection to %v:%v over telnet opened", c.Ip, c.Port)

	}
	if c.Type == SSH {
		lg.Errorf("SSH connections not supported at this time!!!")
		return fmt.Errorf("SSH connections not supported at this time!!!")
	}
	c.Labels["status"] = "connection_opened"
	return nil
}

func (c *Connection) openTelnetConnection() error {
	d := net.Dialer{
		Timeout: conf.ConnTimeout,
	}
	telnet, err := d.Dial("tcp", fmt.Sprintf("%v:%v", c.Ip, c.Port))
	c.conn = telnet
	return err
}

func (c *Connection) Login(login, password, promptName string) error {
	if err, prompt := conf.getPrompt(promptName); err != nil {
		lg.Errorf("Error open connection to %v:%v - %v", c.Ip, c.Port, err.Error())
		return err
	} else {
		c.prompt = prompt
	}
	c.conn.SetDeadline(time.Now().Add(conf.ConnTimeout))

}

func (c *Connection) Command(command, prompt string, addNewLine bool, wait *time.Duration) (error, string) {
	if addNewLine {
		command += "\n"
	}
	c.lastCommandRawData = command
	if wait == nil {
		wait = &(conf.ConnTimeout)
	}
	if err := c.conn.SetDeadline(time.Now().Add(*wait)); err != nil {
		lg.Errorf("Error set deadline timeout for device with IP %v", c.Ip)
		return tracerr.Wrap(err), ""
	}
	if _, err := c.conn.Write([]byte(command)); err != nil {
		lg.Errorf("Error write command '%v' to device with IP %v", command, c.Ip)
		return tracerr.Wrap(err), ""
	}
	b := make([]byte, 1)
	var buf bytes.Buffer
	match, err := regexp.Compile(prompt)
	if err != nil {
		lg.Errorf("Error compile regular prompt '%v' for device with IP %v", prompt, c.Ip)
		return tracerr.Wrap(err), ""
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
		if bytes.Contains([]byte{0x00, 0xFF, 0x02, 0x01, 0x03, 0x04, 0x07, 0x08}, b) {
			continue
		}
		buf.Write(b)
		lines := buf.String()
		for _, line := range strings.Split(buf.String(), "\n") {
			if match.MatchString(line) {
				c.lastCommandRawData = lines
				return nil, lines
			}
		}
	}
}
