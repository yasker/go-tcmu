package main

import (
	"fmt"

	"github.com/yasker/go-tcmu/tcmu"
)

type TcmuHandler interface {
	Name() string
	Subtype() string
	ConfigDesc() string
	CheckConfig(cfgString string) (bool, string)
	Added(dev TcmuDevice)
	Removed(dev TcmuDevice)
}

const (
	DriverName = "Shorthorn"
	Subtype    = "sh"
	ConfigDesc = "dev_config=file/<path>"
)

var (
	finished = make(chan bool)
)

type Handler struct {
}

func (h *Handler) Name() string {
	return DriverName
}

func (h *Handler) Subtype() string {
	return Subtype
}

func (h *Handler) ConfigDesc() string {
	return ConfigDesc
}

func (h *Handler) CheckConfig(cfgString string) (bool, string) {
	return true, ""
}

func (h *Handler) Added(dev TcmuDevice) {
	go processCommands(dev)
}

func (h *Handler) Removed(dev TcmuDevice) {
}

func processCommands(dev *TcmuDevice) {
	for 1 {
		completed := false
		tcmu.ProcessingStart(dev)
		cmd := tcmu.GetNextCommand(dev)
		for cmd != nil {
			ret := handle_cmd(cmd)
			if ret != tcmu.ASYNC_HANDLED {
				tcmu.CommandComplete(dev, cmd, ret)
				completed = true
			}
		}
		if completed {
			tcmu.ProcessingComplete(dev)
		}
		tcmu.WaitForNextEvent(dev)
		cmd = tcmu.GetNextCommand(dev)
	}
}

func main() int {
	var h Handler
	exit := false

	cxt, err := TcmuInit(&h)
	if err != nil {
		panic(err)
	}

	if err := TcmuPollReady(cxt); err != nil {
		panic(err)
	}

	exit := <-finished
	fmt.Println("Finish execution")
}
