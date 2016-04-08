package main

import (
	"io"
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

type server struct{}

func Read(req *block.ReadRequest) (*block.ReadResponse, error) {
	buf := make([]byte, req.Length)
	resp := &block.ReadResponse{
		Result:  "Success",
		Context: buf,
	}
	return resp, nil
}

func Write(req *block.WriteRequest) (*block.WriteResponse, error) {
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
	os.Exit(0)
}

func serve(conn *net.TCPConn) {
	var resp *block.Response

	for {
		req, err := comm.ReadRequest(conn)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error("Fail to read request: ", err)
			return
		}

		attachData := false
		buf := make([]byte, req.Length)
		if req.Type == comm.MSG_TYPE_READ_REQUEST {
			resp = &block.Response{
				Type:   comm.MSG_TYPE_READ_RESPONSE,
				Length: req.Length,
				Result: "Success",
			}
			attachData = true
		} else if req.Type == comm.MSG_TYPE_WRITE_REQUEST {
			buf := make([]byte, req.Length)
			if err := comm.ReceiveData(conn, buf); err != nil {
				log.Error("Fail to receive data:", err)
			}
			resp = &block.Response{
				Type:   comm.MSG_TYPE_WRITE_RESPONSE,
				Result: "Success",
			}
		} else {
			log.Error("Invalid request type: ", req.Type)
			return
		}

		if err := comm.SendResponse(conn, resp); err != nil {
			log.Error("Fail to send response: ", err)
			return
		}
		if attachData {
			if err := comm.SendData(conn, buf); err != nil {
				log.Error("Fail to send data:", err)
			}
		}
	}
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
		go serve(conn)
	}
}
