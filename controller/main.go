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
	"sync"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/yasker/go-tcmu/block"
)

var (
	ready bool = false

	log = logrus.WithFields(logrus.Fields{"pkg": "main"})

	address = "localhost:5000"
)

type TcmuState struct {
	volume    string
	client    block.TransferClient
	conn      *grpc.ClientConn
	lbas      int64
	blockSize int
	mutex     *sync.Mutex
}

//export shOpen
func shOpen(dev TcmuDevice) int {
	var (
		err error
	)

	state := &TcmuState{
		mutex: &sync.Mutex{},
	}
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
	//id := strings.TrimPrefix(cfgString, "file/")
	//TODO check volume name here

	state.conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect to replica, %v", err)
	}

	state.client = block.NewTransferClient(state.conn)

	go state.HandleRequest(dev)

	log.Debugln("Device added")
	return 0
}

func (s *TcmuState) HandleRequest(dev TcmuDevice) {
	defer s.conn.Close()
	for true {
		C.tcmulib_processing_start(dev)
		cmd := C.tcmulib_get_next_command(dev)
		for cmd != nil {
			go s.processCommand(dev, cmd)
			cmd = C.tcmulib_get_next_command(dev)
		}
		ret := C.tcmu_wait_for_next_command(dev)
		if ret != 0 {
			log.Errorln("Fail to wait for next command", ret)
			break
		}
	}
}

func (s *TcmuState) handleReadCommand(dev TcmuDevice, cmd TcmuCommand) int {
	offset := CmdGetLba(cmd) * int64(s.blockSize)
	length := CmdGetXferLength(cmd) * s.blockSize

	resp, err := s.client.Read(context.Background(), &block.ReadRequest{
		Offset: offset,
		Length: int64(length),
	})
	if err != nil {
		log.Errorln("read failed: ", err.Error())
		return CmdSetMediumError(cmd)
	}

	copied := CmdMemcpyIntoIovec(cmd, resp.Context, length)
	if copied != length {
		log.Errorln("read failed: unable to complete buffer copy ")
		return CmdSetMediumError(cmd)
	}
	return C.SAM_STAT_GOOD
}

func (s *TcmuState) handleWriteCommand(dev TcmuDevice, cmd TcmuCommand) int {
	offset := CmdGetLba(cmd) * int64(s.blockSize)
	length := CmdGetXferLength(cmd) * s.blockSize

	buf := make([]byte, length, length)
	if buf == nil {
		log.Errorln("read failed: fail to allocate buffer")
		return CmdSetMediumError(cmd)
	}
	copied := CmdMemcpyFromIovec(cmd, buf, length)
	if copied != length {
		log.Errorln("write failed: unable to complete buffer copy ")
		return CmdSetMediumError(cmd)
	}

	if _, err := s.client.Write(context.Background(), &block.WriteRequest{
		Offset:  offset,
		Context: buf,
	}); err != nil {
		log.Errorln("write failed: ", err.Error())
		return CmdSetMediumError(cmd)
	}

	return C.SAM_STAT_GOOD
}

func (s *TcmuState) processCommand(dev TcmuDevice, cmd TcmuCommand) {
	ret := s.handleCommand(dev, cmd)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	C.tcmulib_command_complete(dev, cmd, C.int(ret))
	C.tcmulib_processing_complete(dev)
}

func (s *TcmuState) handleCommand(dev TcmuDevice, cmd TcmuCommand) int {
	scsiCmd := CmdGetScsiCmd(cmd)
	switch scsiCmd {
	case C.INQUIRY:
		return CmdEmulateInquiry(cmd, dev)
	case C.TEST_UNIT_READY:
		return CmdEmulateTestUnitReady(cmd)
	case C.SERVICE_ACTION_IN_16:
		return CmdEmulateServiceActionIn(cmd, s.lbas, s.blockSize)
	case C.MODE_SENSE, C.MODE_SENSE_10:
		return CmdEmulateModeSense(cmd)
	case C.MODE_SELECT, C.MODE_SELECT_10:
		return CmdEmulateModeSelect(cmd)
	case C.READ_6, C.READ_10, C.READ_12, C.READ_16:
		return s.handleReadCommand(dev, cmd)
	case C.WRITE_6, C.WRITE_10, C.WRITE_12, C.WRITE_16:
		return s.handleWriteCommand(dev, cmd)
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
