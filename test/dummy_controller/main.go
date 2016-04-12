package main

import (
	"flag"
	"net"
	"os"
	"runtime/pprof"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/yasker/longhorn/block"
	"github.com/yasker/longhorn/comm"
)

var (
	log     = logrus.WithFields(logrus.Fields{"pkg": "dummy_controller"})
	address = "localhost:5000"

	size        = flag.Int("size", 1000, "size for read/write, in MB")
	mode        = flag.String("mode", "write", "read or write")
	requestSize = flag.Int("request-size", 4096, "request size of each IO")
	workers     = flag.Int("workers", 128, "worker numbers")

	done     chan bool
	sent     int
	received int

	cpuprofile = "dummy_controller.pf"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	flag.Parse()

	log.Debug("Output cpuprofile to %v", cpuprofile)
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	if *mode != "read" && *mode != "write" {
		log.Fatal("Invalid mode type ", *mode)
	}

	log.Infof("Mode %v, size %vMB, request size %v bytes, %v workers\n",
		*mode, *size, *requestSize, *workers)

	addr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		log.Fatalf("failed to resolve ", address, err)
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Fatalf("Cannot connect to replica, %v", err)
	}
	defer conn.Close()

	log.Info("Start processing")

	processData(conn)

	log.Info("Finish processing")
}

func processRequest(conn *net.TCPConn, requests chan *block.Request) {
	for req := range requests {
		if err := comm.SendRequest(conn, req); err != nil {
			log.Error("Fail to send request:", err)
			continue
		}
		if *mode == "read" {
			if req.Type != comm.MSG_TYPE_READ_REQUEST {
				log.Error("Wrong kinds of request: ", req.Type)
				continue
			}
		} else {
			if req.Type != comm.MSG_TYPE_WRITE_REQUEST {
				log.Error("Wrong kinds of request: ", req.Type)
				continue
			}
		}
		if req.Type == comm.MSG_TYPE_WRITE_REQUEST {
			buf := make([]byte, req.Length, req.Length)
			if err := comm.SendData(conn, buf); err != nil {
				log.Error("Fail to send data:", err)
				continue
			}
		}
		sent++
	}
}

func processResponse(conn *net.TCPConn) {
	for {
		resp, err := comm.ReadResponse(conn)
		if err != nil {
			break
			log.Error("Fail to read response:", err)
			continue
		}
		if resp.Result != "Success" {
			log.Error("Operation failed: ", resp.Result)
			continue
		}
		if *mode == "read" {
			if resp.Type != comm.MSG_TYPE_READ_RESPONSE {
				log.Error("Wrong kinds of response: ", resp.Type)
				continue
			}
		} else {
			if resp.Type != comm.MSG_TYPE_WRITE_RESPONSE {
				log.Error("Wrong kinds of response: ", resp.Type)
				continue
			}
		}
		if resp.Type == comm.MSG_TYPE_READ_RESPONSE {
			buf := make([]byte, resp.Length, resp.Length)
			if err := comm.ReceiveData(conn, buf); err != nil {
				log.Error("Receive data failed:", err)
				continue
			}
		}
		received++
	}
}

func processData(conn *net.TCPConn) {
	before := time.Now()
	reqSize := int64(*requestSize)
	sizeInBytes := int64(*size * 1024 * 1024)

	requests := make(chan *block.Request, 16)
	go processRequest(conn, requests)
	go processResponse(conn)

	for offset := int64(0); offset < sizeInBytes-reqSize; offset += reqSize {
		if offset%(1024*1024*100) == 0 {
			log.Debug("Processing offset ", offset)
		}

		if *mode == "write" {
			requests <- &block.Request{
				Type:   comm.MSG_TYPE_WRITE_REQUEST,
				Offset: offset,
				Length: reqSize,
			}
		} else {
			requests <- &block.Request{
				Type:   comm.MSG_TYPE_READ_REQUEST,
				Offset: offset,
				Length: reqSize,
			}
		}
	}
	close(requests)
	for sent != received {
		time.Sleep(10 * time.Millisecond)
	}

	seconds := time.Now().Sub(before).Seconds()
	bandwidth := float64(sizeInBytes) / seconds / 1024 / 1024
	iops := sizeInBytes / int64(seconds) / reqSize
	log.Debugf("Processing done, speed at %.2f MB/second, %v request/seconds",
		bandwidth, iops)
}

/*
func processData(client block.TransferClient) {
	before := time.Now()
	reqSize := int64(*requestSize)

	co := make(chan int64, *workers)
	wg := sync.WaitGroup{}
	wg.Add(*workers)
	for i := 0; i < *workers; i++ {
		go func() {
			defer wg.Done()
			process(client, *mode, reqSize, co)
		}()
	}

	sizeInBytes := int64(*size * 1024 * 1024)
	for offset := int64(0); offset < sizeInBytes-reqSize; offset += reqSize {
		co <- offset
	}
	close(co)
	wg.Wait()

	seconds := time.Now().Sub(before).Seconds()
	bandwidth := float64(sizeInBytes) / seconds / 1024 / 1024
	iops := sizeInBytes / int64(seconds) / reqSize
	log.Debugf("Processing done, speed at %.2f MB/second, %v request/seconds",
		bandwidth, iops)
}

func process(client block.TransferClient, mode string, reqSize int64, co chan int64) {
	for offset := range co {
		var err error
		if offset%(1024*1024*100) == 0 {
			log.Debug("Processing offset ", offset)
		}

		if mode == "write" {
			buf := make([]byte, reqSize, reqSize)
			_, err = client.Write(context.Background(), &block.WriteRequest{
				Offset:  offset,
				Context: buf,
			})
		} else {
			var resp *block.ReadResponse
			buf := make([]byte, reqSize, reqSize)
			resp, err = client.Read(context.Background(), &block.ReadRequest{
				Offset: offset,
				Length: reqSize,
			})
			copy(buf, resp.Context)
		}
		if err != nil {
			log.Errorln("Fail to process data from offset ", offset)
		}
	}
}
*/
