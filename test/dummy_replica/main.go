package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/Sirupsen/logrus"

	"github.com/yasker/longhorn/block"
	"github.com/yasker/longhorn/comm"
)

const (
	port = ":5000"
	size = 1073741824
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "dummy_replica"})

	cpuprofile = "dummy_replica.pf"

	sigs chan os.Signal
	done bool
)

func handleSignal() {
	sig := <-sigs
	log.Infoln("Shutting down process, due to received signal ", sig)
	pprof.StopCPUProfile()
	os.Exit(0)
}

func RequestHandler(req *comm.Request) (*comm.Response, error) {
	if req.Header.Type == comm.MSG_TYPE_READ_REQUEST {
		return &comm.Response{
			Header: &block.Response{
				Id:     req.Header.Id,
				Type:   comm.MSG_TYPE_READ_RESPONSE,
				Length: req.Header.Length,
				Result: "Success",
			},
			Data: make([]byte, req.Header.Length),
		}, nil
	}
	if req.Header.Type == comm.MSG_TYPE_WRITE_REQUEST {
		return &comm.Response{
			Header: &block.Response{
				Id:     req.Header.Id,
				Type:   comm.MSG_TYPE_WRITE_RESPONSE,
				Result: "Success",
			},
		}, nil
	}
	return nil, fmt.Errorf("Invalid request type: ", req.Header.Type)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	sigs = make(chan os.Signal, 1)
	done = false

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go handleSignal()

	log.Debug("Output cpuprofile to %v", cpuprofile)
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)

	addr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		log.Fatalf("failed to resolve ", port, err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen to: %v", err)
	}

	for !done {
		conn, err := l.AcceptTCP()
		if err != nil {
			log.Errorf("failed to accept connection %v", err)
			continue
		}
		server := comm.NewServer(conn, 128, RequestHandler)
		server.Start()
	}
}
