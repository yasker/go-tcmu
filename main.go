package main

/*
#cgo LDFLAGS: -L ./libs -ltcmu
#cgo CFLAGS: -I ./includes

#include <errno.h>
#include <stdlib.h>
#include "libtcmu.h"

extern struct tcmulib_context *tcmu_init();
extern bool tcmu_poll_master_fd(struct tcmulib_context *cxt);
*/
import "C"
import "unsafe"

import (
	"github.com/Sirupsen/logrus"
)

var (
	ready bool = false

	log = logrus.WithFields(logrus.Fields{"pkg": "main"})
)

//export shOpen
func shOpen(dev *C.struct_tcmu_device) int {
	blockSizeStr := C.CString("hw_block_size")
	defer C.free(unsafe.Pointer(blockSizeStr))
	blockSize := C.tcmu_get_attribute(dev, blockSizeStr)
	if blockSize == -1 {
		log.Error("Cannot find valid hw_block_size")
		return -C.EINVAL
	}
	size := C.tcmu_get_device_size(dev)
	if size == -1 {
		log.Error("Cannot find valid disk size")
		return -C.EINVAL
	}

	cfgString := C.tcmu_get_dev_cfgstring(dev)
	log.Debugln("Config string is ", C.GoString(cfgString))
	log.Debugf("Size is %v, blocksize is %v", size, blockSize)
	return -C.EINVAL
}

//export shClose
func shClose(dev *C.struct_tcmu_device) {
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
