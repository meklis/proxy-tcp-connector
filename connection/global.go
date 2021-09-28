package connection

import (
	"github.com/meklis/http-snmpwalk-proxy/logger"
	"os"
	"time"
)

type Config struct {
	ConnTimeout          time.Duration `yaml:"conn_timeout"`
	Prompts              Prompt        `yaml:"prompts"`
	AfterLoginCommands   []string      `yaml:"after_login_commands"`
	BeforeLogoutCommands []string      `yaml:"before_logout_commands"`
}

type Prompt struct {
	Command  string `yaml:"command"`
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
}

var lg = getDefaultLogger()

func SetLogger(log *logger.Logger) {
	lg = log
}

func getDefaultLogger() *logger.Logger {
	log, _ := logger.New("no_log", 0, os.DevNull)
	return log
}
