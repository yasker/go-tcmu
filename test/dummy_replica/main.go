package main

import (
	"net"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/yasker/longhorn/block"
)

const (
	port = ":5000"
	size = 1073741824
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "replica"})
)

type server struct{}

func (s *server) Read(cxt context.Context, req *block.ReadRequest) (*block.ReadResponse, error) {
	buf := make([]byte, req.Length)
	resp := &block.ReadResponse{
		Result:  "Success",
		Context: buf,
	}
	return resp, nil
}

func (s *server) Write(cxt context.Context, req *block.WriteRequest) (*block.WriteResponse, error) {
	buf := make([]byte, len(req.Context))
	copy(buf, req.Context)
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

	block.RegisterTransferServer(s, server)
	s.Serve(l)
}
