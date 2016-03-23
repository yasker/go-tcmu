package tcmu

// #cgo LDFLAGS: -I ./includes -L ./libs -ltcmu
//
//#include <stdio.h>
//#include <stdlib.h>
//#include <stdarg.h>
//#include "../includes/libtcmu.h"
//void errp(const char *fmt, ...)
//{
//	va_list va;
//
//	va_start(va, fmt);
//	vfprintf(stderr, fmt, va);
//	va_end(va);
//}
//
import "C"

import "unsafe"

const (
	TcmuNotHandled   = C.TCMU_NOT_HANDLED
	TcmuAsyncHandled = C.TCMU_ASYNC_HANDLED
)

type (
	CTcmuHandler C.struct_tcmulib_handler
	CTcmuContext C.struct_tcmulib_context
	CTcmuCmd     C.struct_tcmulib_cmd
	CTcmuDevice  C.struct_tcmu_device
)

//func tcmuInitialize()

func GetMasterFd(cxt *CTcmuContext) int {
	return int(C.tcmulib_get_master_fd((*C.struct_tcmulib_context)(cxt)))
}

func MasterFdReady(cxt *CTcmuContext) int {
	return int(C.tcmulib_master_fd_ready((*C.struct_tcmulib_context)(cxt)))
}

func GetNextCommand(dev *CTcmuDevice) *CTcmuCmd {
	return nil
}

func CommandComplete(dev *CTcmuDevice, cmd *CTcmuCmd, result int) {
	C.tcmulib_command_complete((*C.struct_tcmu_device)(dev), (*C.struct_tcmulib_cmd)(cmd), (C.int)(result))
}

func ProcessingStart(dev *CTcmuDevice) {
	C.tcmulib_processing_start((*C.struct_tcmu_device)(dev))
}

func ProcessingComplete(dev *CTcmuDevice) {
	C.tcmulib_processing_complete((*C.struct_tcmu_device)(dev))
}

func Close(cxt *CTcmuContext) {
	C.tcmulib_close((*C.struct_tcmulib_context)(cxt))
}

//func GetDevPrivate()
//func SetDevPrivate()

func GetDevFd(dev *CTcmuDevice) int {
	return int(C.tcmu_get_dev_fd((*C.struct_tcmu_device)(dev)))
}

func GetDevCfgstring(dev *CTcmuDevice) string {
	ret := (*C.char)(C.tcmu_get_dev_cfgstring((*C.struct_tcmu_device)(dev)))
	defer C.free(unsafe.Pointer(ret))

	return C.GoString(ret)
}

//func tcmuGetDeviceHandler

func GetAttribute(dev *CTcmuDevice, name string) int {
	var cName *C.char = C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return (int)(C.tcmu_get_attribute((*C.struct_tcmu_device)(dev), cName))
}

func GetDeviceSize(dev *CTcmuDevice) int64 {
	return (int64)(C.tcmu_get_device_size((*C.struct_tcmu_device)(dev)))
}

//func tcmuGetCdbLength
//func tcmuGetLba
//func tcmuGetXferLength
