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

extern uint8_t tcmucmd_get_scsi_cmd(struct tcmulib_cmd *cmd);

extern int tcmucmd_emulate_inquiry(struct tcmulib_cmd *cmd, struct tcmu_device *dev);
extern int tcmucmd_emulate_test_unit_ready(struct tcmulib_cmd *cmd);
extern tcmucmd_emulate_service_action_in(struct tcmulib_cmd *cmd,
		uint64_t num_lbas, uint32_t block_size);
extern int tcmucmd_emulate_mode_sense(struct tcmulib_cmd *cmd);
extern int tcmucmd_emulate_mode_select(struct tcmulib_cmd *cmd);
extern int tcmucmd_set_medium_error(struct tcmulib_cmd *cmd);
extern uint64_t tcmucmd_get_lba(struct tcmulib_cmd *cmd);
extern uint32_t tcmucmd_get_xfer_length(struct tcmulib_cmd *cmd);
extern void *allocate_buffer(int length);
extern int tcmucmd_memcpy_into_iovec(struct tcmulib_cmd *cmd, void *buf, int length);
extern int tcmucmd_memcpy_from_iovec(struct tcmulib_cmd *cmd, void *buf, int length);

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
func shOpen(dev *C.struct_tcmu_device) int {
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
	scsiCmd := C.tcmucmd_get_scsi_cmd(cmd)
	switch scsiCmd {
	case C.INQUIRY:
		return int(C.tcmucmd_emulate_inquiry(cmd, dev))
	case C.TEST_UNIT_READY:
		return int(C.tcmucmd_emulate_test_unit_ready(cmd))
	case C.SERVICE_ACTION_IN_16:
		return int(C.tcmucmd_emulate_service_action_in(cmd, C.uint64_t(state.lbas), C.uint32_t(state.blockSize)))
	case C.MODE_SENSE, C.MODE_SENSE_10:
		return int(C.tcmucmd_emulate_mode_sense(cmd))
	case C.MODE_SELECT, C.MODE_SELECT_10:
		return int(C.tcmucmd_emulate_mode_select(cmd))
	case C.READ_6, C.READ_10, C.READ_12, C.READ_16:
		offset := int64(C.tcmucmd_get_lba(cmd)) * int64(state.blockSize)
		length := int(C.tcmucmd_get_xfer_length(cmd)) * state.blockSize

		buf := C.allocate_buffer(C.int(length))
		if buf == nil {
			log.Errorln("read failed: fail to allocate buffer")
			return int(C.tcmucmd_set_medium_error(cmd))
		}

		goBuf := C.GoBytes(buf, C.int(length))
		defer C.free(buf)
		if readed, err := state.file.ReadAt(goBuf, offset); err != nil {
			if readed != length || err != io.EOF {
				log.Errorln("read failed: ", err.Error())
				return int(C.tcmucmd_set_medium_error(cmd))
			}
		}

		copied := C.tcmucmd_memcpy_into_iovec(cmd, buf, C.int(length))
		if int(copied) != length {
			log.Errorln("read failed: unable to complete buffer copy ")
			return int(C.tcmucmd_set_medium_error(cmd))
		}

		return C.SAM_STAT_GOOD
	case C.WRITE_6, C.WRITE_10, C.WRITE_12, C.WRITE_16:
		offset := int64(C.tcmucmd_get_lba(cmd)) * int64(state.blockSize)
		length := int(C.tcmucmd_get_xfer_length(cmd)) * state.blockSize

		buf := C.allocate_buffer(C.int(length))
		if buf == nil {
			log.Errorln("read failed: fail to allocate buffer")
			return int(C.tcmucmd_set_medium_error(cmd))
		}
		copied := C.tcmucmd_memcpy_from_iovec(cmd, buf, C.int(length))
		if int(copied) != length {
			log.Errorln("write failed: unable to complete buffer copy ")
			return int(C.tcmucmd_set_medium_error(cmd))
		}

		goBuf := C.GoBytes(buf, C.int(length))
		defer C.free(buf)
		if _, err := state.file.WriteAt(goBuf, offset); err != nil {
			log.Errorln("write failed: ", err.Error())
			return int(C.tcmucmd_set_medium_error(cmd))
		}

		return C.SAM_STAT_GOOD
	default:
		log.Errorf("unknown command 0x%x\n", scsiCmd)
	}
	return C.TCMU_NOT_HANDLED
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
