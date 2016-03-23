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
	go h.processCommands(dev)
}

func (h *Handler) Removed(dev TcmuDevice) {
}

func (h *Handler) processCommands(dev *TcmuDevice) {
	for 1 {
		completed := false
		tcmu.ProcessingStart(dev)
		cmd := tcmu.GetNextCommand(dev)
		for cmd != nil {
			ret := h.handleCommand(dev, cmd)
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

func (h *Handler) handleCommand(dev *TcmuDevice, cmd *TcmuCmd) int {
	cdb := cmd.Cdb
	iovec := cmd.Iovec
	iovCnt := cmd.IovCnt
	sense := cmd.SenseBuf
	fileState := h.State
	ret = 0

	scsiCmd := cdb[0]
	switch scsiCmd {
	case tcmu.INQUIRY:
		return tcmu_emulate_inquiry(dev, cdb, iovec, iov_cnt, sense)
	case tcmu.TEST_UNIT_READY:
		return tcmu_emulate_test_unit_ready(cdb, iovec, iov_cnt, sense)
	case tcmu.SERVICE_ACTION_IN_16:
		if cdb[1] == READ_CAPACITY_16 {
			return tcmu.EmulateReadCapacity_16(state.NumLbas,
				state.BlockSize,
				cdb, iovec, iov_cnt, sense)
		}
		return TCMU_NOT_HANDLED
	case tcmu.MODE_SENSE:
	case tcmu.MODE_SENSE_10:
		return tcmu.EmulateModeSense(cdb, iovec, iov_cnt, sense)
	case tcmu.MODE_SELECT:
	case tcmu.MODE_SELECT_10:
		return tcmu.EmulateModeSelect(cdb, iovec, iov_cnt, sense)
	case tcmu.READ_6:
	case tcmu.READ_10:
	case tcmu.READ_12:
	case tcmu.READ_16:
		offset := state.BlockSize * tcmu.CdbGetLba(cdb)
		length := state.BlockSize * tcmu.CdbGetXferLength(cdb)
		buf = make([]byte, length)
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			log.Error("Fail to read due to %s", err.Error())
			return tcmu.SetMediumError(sense)
		}
		tcmu.CopyIntoIovec(iovec, iovCnt, buf, length)
		return tcmu.SAM_STAT_GOOD
	case tcmu.WRITE_6:
	case tcmu.WRITE_10:
	case tcmu.WRITE_12:
	case tcmu.WRITE_16:
		offset := state.BlockSize * tcmu.CdbGetLba(cdb)
		length := state.BlockSize * tcmu.CdbGetXferLength(cdb)
		remaining := length
		for iov := range iovec {
			toCopy = remaining
			if iov.Len < remaining {
				toCopy = iov.Len
			}
			_, err := f.WriteAt(iov.Base, offset)
			if err != nil {
				log.Error("Fail to write due to %s", err.Error())
				return tcmu.SetMediumError(sense)
			}
			offset += toCopy
		}
		return tcmu.SAM_STAT_GOOD
	default:
		log.Error("Unknown command 0x%x", scsiCmd)
	}
	return TCMU_NOT_HANDLED
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
