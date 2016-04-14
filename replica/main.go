package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Sirupsen/logrus"

	"github.com/yasker/longhorn/block"
	"github.com/yasker/longhorn/rpc"
	"github.com/yasker/longhorn/util"
)

const (
	port     = ":5000"
	filename = "test.img"
	size     = 1073741824
)

var (
	log  = logrus.WithFields(logrus.Fields{"pkg": "replica"})
	file *os.File
)

func RequestHandler(req *rpc.Request) (*rpc.Response, error) {
	if file == nil {
		return nil, fmt.Errorf("File is not ready")
	}
	if req.Header.Type == rpc.MSG_TYPE_READ_REQUEST {
		buf := make([]byte, req.Header.Length)
		if _, err := file.ReadAt(buf, req.Header.Offset); err != nil && err != io.EOF {
			log.Errorln("read failed: ", err.Error())
			return nil, err
		}
		return &rpc.Response{
			Header: &block.Response{
				Id:     req.Header.Id,
				Type:   rpc.MSG_TYPE_READ_RESPONSE,
				Length: req.Header.Length,
				Result: "Success",
			},
			Data: buf,
		}, nil
	}
	if req.Header.Type == rpc.MSG_TYPE_WRITE_REQUEST {
		if _, err := file.WriteAt(req.Data, req.Header.Offset); err != nil {
			log.Errorln("write failed: ", err.Error())
			return nil, err
		}
		return &rpc.Response{
			Header: &block.Response{
				Id:     req.Header.Id,
				Type:   rpc.MSG_TYPE_WRITE_RESPONSE,
				Result: "Success",
			},
		}, nil
	}
	return nil, fmt.Errorf("Invalid request type: ", req.Header.Type)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	addr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		log.Fatalf("failed to resolve ", port, err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen to: %v", err)
	}

	if err := util.FindOrCreateDisk(filename, size); err != nil {
		log.Fatalf("Fail to find or create disk", err.Error())
	}
	file, err = os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Fail to open disk file", err.Error())
	}

	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			log.Errorf("failed to accept connection %v", err)
			continue
		}
		server := rpc.NewServer(conn, 128, RequestHandler)
		server.Start()
	}
}
