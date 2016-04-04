package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/yasker/longhorn/block"
	"github.com/yasker/longhorn/util"
)

const (
	port     = ":5000"
	filename = "test.img"
	size     = 1073741824
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "replica"})
)

type server struct {
	file *os.File
}

func (s *server) Read(cxt context.Context, req *block.ReadRequest) (*block.ReadResponse, error) {
	if s.file == nil {
		return nil, fmt.Errorf("File is not ready")
	}
	buf := make([]byte, req.Length)
	if _, err := s.file.ReadAt(buf, req.Offset); err != nil && err != io.EOF {
		log.Errorln("read failed: ", err.Error())
		return nil, err
	}
	resp := &block.ReadResponse{
		Result:  "Success",
		Context: buf,
	}
	return resp, nil
}

func (s *server) Write(cxt context.Context, req *block.WriteRequest) (*block.WriteResponse, error) {
	if s.file == nil {
		return nil, fmt.Errorf("File is not ready")
	}
	if _, err := s.file.WriteAt(req.Context, req.Offset); err != nil {
		log.Errorln("write failed: ", err.Error())
		return nil, err
	}
	resp := &block.WriteResponse{
		Result: "Success",
	}

	return resp, nil
}

func main() {
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen to: %v", err)
	}
	s := grpc.NewServer()

	server := &server{}
	if err := util.FindOrCreateDisk(filename, size); err != nil {
		log.Fatalf("Fail to find or create disk", err.Error())
	}
	server.file, err = os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Fail to open disk file", err.Error())
	}

	block.RegisterTransferServer(s, server)
	s.Serve(l)
}
