package main

import (
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

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
	log = logrus.WithFields(logrus.Fields{"pkg": "dummy_replica"})

	cpuprofile = "dummy_replica.pf"

	sigs chan os.Signal
	done chan bool
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

func handleSignal() {
	sig := <-sigs
	log.Infoln("Shutting down process, due to received signal ", sig)
	pprof.StopCPUProfile()
	done <- true
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	sigs = make(chan os.Signal, 1)
	done = make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go handleSignal()

	log.Debug("Output cpuprofile to %v", cpuprofile)
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)

	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen to: %v", err)
	}

	s := grpc.NewServer()
	server := &server{}

	block.RegisterTransferServer(s, server)
	go s.Serve(l)

	<-done
}
