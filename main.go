package main

/*
#cgo LDFLAGS: -L ./libs -ltcmu
#cgo CFLAGS: -I ./includes

#include <errno.h>
#include <stdlib.h>
#include "libtcmu.h"

extern struct tcmulib_context *tcmu_init();
extern bool tcmu_poll_master_fd(struct tcmulib_context *cxt);
extern int tcmu_wait_for_next_command(struct tcmu_device *dev);

*/
import "C"
import "unsafe"

import (
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
	blockSize int64
}

//export shOpen
func shOpen(dev *C.struct_tcmu_device) int {
	var (
		state TcmuState
		err   error
	)

	blockSizeStr := C.CString("hw_block_size")
	defer C.free(unsafe.Pointer(blockSizeStr))
	blockSize := int64(C.tcmu_get_attribute(dev, blockSizeStr))
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
	state.lbas = size / state.blockSize

	cfgString := C.GoString(C.tcmu_get_dev_cfgstring(dev))
	if cfgString == "" {
		log.Errorln("Cannot find configuration string")
		return -C.EINVAL
	}
	path := strings.TrimLeft(cfgString, "file:/")

	if err := util.FindOrCreateDisk(path, size); err != nil {
		log.Errorln("Fail to find or create disk", err.Error())
		return -C.EINVAL
	}
	state.file, err = os.Open(path)
	if err != nil {
		log.Errorln("Fail to open disk file", err.Error())
		return -C.EINVAL
	}
	go handleRequest(dev, &state)

	log.Debugln("Device added")
	return 0
}

func handleRequest(dev *C.struct_tcmu_device, state *TcmuState) {
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

func handleCommand(dev *C.struct_tcmu_device, cmd *C.struct_tcmulib_cmd, state *TcmuState) int {
	return C.SAM_STAT_GOOD
}

//export shClose
func shClose(dev *C.struct_tcmu_device) {
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
