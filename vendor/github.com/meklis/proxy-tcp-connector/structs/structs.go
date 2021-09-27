package structs

import (
	"net"
	"time"
)

type (
	Device struct {
		Ip      string
		Port    int
		Timeout time.Duration
		Uid     string
		Login   string
		Password string
	}

	CloseConnect struct {
		Ip  string
		Uid string
	}

	ControlStruct struct {
		Err     error
		Code    int
		Module  string
		Message string
	}


	Connection struct {
		NetConn *net.Conn
		Device
	}
)
