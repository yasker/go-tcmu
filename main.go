package main

/*
#cgo LDFLAGS: -L ./libs -ltcmu
#cgo CFLAGS: -I ./includes

#include <errno.h>
#include <stdlib.h>
#include <scsi/scsi.h>
#include "libtcmu.h"

extern struct tcmulib_context *tcmu_init();
extern bool tcmu_poll_master_fd(struct tcmulib_context *cxt);
extern int tcmu_wait_for_next_command(struct tcmu_device *dev);
extern void *allocate_buffer(int length);

*/
import "C"
import "unsafe"

import (
	"io"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/yasker/go-tcmu/util"
)

var (
	ready bool = false

	log = logrus.WithFields(logrus.Fields{"pkg": "main"})
)

type TcmuState struct {
	file      *os.File
	lbas      int64
	blockSize int
}

//export shOpen
func shOpen(dev TcmuDevice) int {
	var (
		state TcmuState
		err   error
	)

	blockSizeStr := C.CString("hw_block_size")
	defer C.free(unsafe.Pointer(blockSizeStr))
	blockSize := int(C.tcmu_get_attribute(dev, blockSizeStr))
	if blockSize == -1 {
		log.Errorln("Cannot find valid hw_block_size")
		return -C.EINVAL
	}
	state.blockSize = blockSize

	size := int64(C.tcmu_get_device_size(dev))
	if size == -1 {
		log.Errorln("Cannot find valid disk size")
		return -C.EINVAL
	}
	state.lbas = size / int64(state.blockSize)

	cfgString := C.GoString(C.tcmu_get_dev_cfgstring(dev))
	if cfgString == "" {
		log.Errorln("Cannot find configuration string")
		return -C.EINVAL
	}
	path := strings.TrimPrefix(cfgString, "file/")
	log.Debugln("File at ", path)

	if err := util.FindOrCreateDisk(path, size); err != nil {
		log.Errorln("Fail to find or create disk", err.Error())
		return -C.EINVAL
	}
	state.file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		log.Errorln("Fail to open disk file", err.Error())
		return -C.EINVAL
	}
	go handleRequest(dev, &state)

	log.Debugln("Device added")
	return 0
}

func handleRequest(dev TcmuDevice, state *TcmuState) {
	defer state.file.Close()
	for true {
		completed := false

		C.tcmulib_processing_start(dev)
		cmd := C.tcmulib_get_next_command(dev)
		for cmd != nil {
			ret := handleCommand(dev, cmd, state)
			if ret != C.TCMU_ASYNC_HANDLED {
				C.tcmulib_command_complete(dev, cmd, C.int(ret))
				completed = true
			}
			cmd = C.tcmulib_get_next_command(dev)
		}
		if completed {
			C.tcmulib_processing_complete(dev)
		}
		ret := C.tcmu_wait_for_next_command(dev)
		if ret != 0 {
			log.Errorln("Fail to wait for next command", ret)
			break
		}
	}
}

func handleCommand(dev TcmuDevice, cmd TcmuCommand, state *TcmuState) int {
	scsiCmd := CmdGetScsiCmd(cmd)
	switch scsiCmd {
	case C.INQUIRY:
		return CmdEmulateInquiry(cmd, dev)
	case C.TEST_UNIT_READY:
		return CmdEmulateTestUnitReady(cmd)
	case C.SERVICE_ACTION_IN_16:
		return CmdEmulateServiceActionIn(cmd, state.lbas, state.blockSize)
	case C.MODE_SENSE, C.MODE_SENSE_10:
		return CmdEmulateModeSense(cmd)
	case C.MODE_SELECT, C.MODE_SELECT_10:
		return CmdEmulateModeSelect(cmd)
	case C.READ_6, C.READ_10, C.READ_12, C.READ_16:
		offset := CmdGetLba(cmd) * int64(state.blockSize)
		length := CmdGetXferLength(cmd) * state.blockSize

		buf := C.allocate_buffer(C.int(length))
		if buf == nil {
			log.Errorln("read failed: fail to allocate buffer")
			return CmdSetMediumError(cmd)
		}

		goBuf := (*[1 << 30]byte)(unsafe.Pointer(buf))[:length:length]
		defer C.free(buf)
		if _, err := state.file.ReadAt(goBuf, offset); err != nil && err != io.EOF {
			log.Errorln("read failed: ", err.Error())
			return CmdSetMediumError(cmd)
		}

		copied := CmdMemcpyIntoIovec(cmd, buf, length)
		if copied != length {
			log.Errorln("read failed: unable to complete buffer copy ")
			return CmdSetMediumError(cmd)
		}

		return C.SAM_STAT_GOOD
	case C.WRITE_6, C.WRITE_10, C.WRITE_12, C.WRITE_16:
		offset := CmdGetLba(cmd) * int64(state.blockSize)
		length := CmdGetXferLength(cmd) * state.blockSize

		buf := C.allocate_buffer(C.int(length))
		if buf == nil {
			log.Errorln("read failed: fail to allocate buffer")
			return CmdSetMediumError(cmd)
		}
		copied := CmdMemcpyFromIovec(cmd, buf, length)
		if copied != length {
			log.Errorln("write failed: unable to complete buffer copy ")
			return CmdSetMediumError(cmd)
		}

		goBuf := (*[1 << 30]byte)(unsafe.Pointer(buf))[:length:length]
		defer C.free(buf)
		if _, err := state.file.WriteAt(goBuf, offset); err != nil {
			log.Errorln("write failed: ", err.Error())
			return CmdSetMediumError(cmd)
		}

		return C.SAM_STAT_GOOD
	default:
		log.Errorf("unknown command 0x%x\n", scsiCmd)
	}
	return C.TCMU_NOT_HANDLED
}

//export shClose
func shClose(dev TcmuDevice) {
	log.Debugln("Device removed")
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	cxt := C.tcmu_init()
	if cxt == nil {
		panic("cxt is nil")
	}

	for !ready {
		result := C.tcmu_poll_master_fd(cxt)
		log.Debugln("Poll master fd one more time, last result ", result)
	}
}
