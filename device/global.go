package device

import (
	"fmt"
	"github.com/meklis/http-snmpwalk-proxy/logger"
	"os"
	"time"
)

type Config struct {
	ConnTimeout time.Duration     `yaml:"conn_timeout"`
	Prompts     map[string]Prompt `yaml:"prompts"`
}

type Prompt struct {
	Command  string `yaml:"command"`
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

var conf = &Config{}
var lg = getDefaultLogger()

func SetLogger(log *logger.Logger) {
	lg = log
}

func SetConfig(cnf *Config) {
	conf = cnf
}

func (c *Config) getPrompt(prompt string) (error, *Prompt) {
	if d, exist := c.Prompts[prompt]; exist {
		return nil, &d
	} else {
		return fmt.Errorf("prompt '%v' not found in configuration", prompt), nil
	}
}

func getDefaultLogger() *logger.Logger {
	log, _ := logger.New("no_log", 0, os.DevNull)
	return log
}
