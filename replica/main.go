package main

import (
	"net"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/yasker/go-tcmu/block"
)

const (
	port = ":5000"
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "replica"})
)

type server struct{}

func (s *server) Read(cxt context.Context, req *block.ReadRequest) (*block.ReadResponse, error) {
	return nil, nil
}

func (s *server) Write(cxt context.Context, req *block.WriteRequest) (*block.WriteResponse, error) {
	return nil, nil
}

func main() {
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen to: %v", err)
	}
	s := grpc.NewServer()
	block.RegisterTransferServer(s, &server{})
	s.Serve(l)
}
