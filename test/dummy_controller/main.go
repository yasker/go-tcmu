package main

import (
	"flag"
	"net"
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
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

	idChanMap      map[int64]chan *WholeResponse
	idCounter      int64 = 0
	idChanMapMutex *sync.Mutex
)

type WholeRequest struct {
	header *block.Request
	data   []byte
}

type WholeResponse struct {
	header *block.Response
	data   []byte
}

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

	idChanMap = make(map[int64]chan *WholeResponse)
	idChanMapMutex = &sync.Mutex{}

	log.Info("Start processing")

	processData(conn)

	log.Info("Finish processing")
}

func processRequest(conn *net.TCPConn, requests chan *WholeRequest) {
	for req := range requests {
		header := req.header
		if err := comm.SendRequest(conn, header); err != nil {
			log.Error("Fail to send request:", err)
			continue
		}
		if *mode == "read" {
			if header.Type != comm.MSG_TYPE_READ_REQUEST {
				log.Error("Wrong kinds of request: ", header.Type)
				continue
			}
		} else {
			if header.Type != comm.MSG_TYPE_WRITE_REQUEST {
				log.Error("Wrong kinds of request: ", header.Type)
				continue
			}
		}
		if header.Type == comm.MSG_TYPE_WRITE_REQUEST {
			if err := comm.SendData(conn, req.data); err != nil {
				log.Error("Fail to send data:", err)
				continue
			}
		}
		sent++
	}
}

func processResponse(conn *net.TCPConn) {
	for {
		var (
			response *WholeResponse
			data     []byte
		)
		respHeader, err := comm.ReadResponse(conn)
		if err != nil {
			break
			log.Error("Fail to read response:", err)
			continue
		}
		if respHeader.Result != "Success" {
			log.Error("Operation failed: ", respHeader.Result)
			continue
		}
		if *mode == "read" {
			if respHeader.Type != comm.MSG_TYPE_READ_RESPONSE {
				log.Error("Wrong kinds of response: ", respHeader.Type)
				continue
			}
		} else {
			if respHeader.Type != comm.MSG_TYPE_WRITE_RESPONSE {
				log.Error("Wrong kinds of response: ", respHeader.Type)
				continue
			}
		}
		if respHeader.Type == comm.MSG_TYPE_READ_RESPONSE {
			data = make([]byte, respHeader.Length, respHeader.Length)
			if err := comm.ReceiveData(conn, data); err != nil {
				log.Error("Receive data failed:", err)
				continue
			}
		}
		idChanMapMutex.Lock()
		respChan := idChanMap[respHeader.Id]
		delete(idChanMap, respHeader.Id)
		idChanMapMutex.Unlock()
		response = &WholeResponse{
			header: respHeader,
			data:   data,
		}
		respChan <- response
		received++
	}
}

/*
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
	bandwidthBits := bandwidth * 8
	iops := float64(sizeInBytes) / float64(reqSize) / float64(seconds)
	log.Debugf("Processing done, speed at %.2f MB/sec(%.2f Mb/sec), %.0f request/seconds",
		bandwidth, bandwidthBits, iops)
}
*/

func processData(conn *net.TCPConn) {
	before := time.Now()
	reqSize := int64(*requestSize)

	requests := make(chan *WholeRequest, *workers)
	go processRequest(conn, requests)
	go processResponse(conn)

	co := make(chan int64, *workers)
	wg := sync.WaitGroup{}
	wg.Add(*workers)
	for i := 0; i < *workers; i++ {
		go func() {
			defer wg.Done()
			process(conn, requests, *mode, reqSize, co)
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
	bandwidthBits := bandwidth * 8
	iops := float64(sizeInBytes) / float64(reqSize) / float64(seconds)
	log.Debugf("Processing done, speed at %.2f MB/sec(%.2f Mb/sec), %.0f request/seconds",
		bandwidth, bandwidthBits, iops)
}

func process(conn *net.TCPConn, requests chan *WholeRequest, mode string, reqSize int64, co chan int64) {
	for offset := range co {
		var (
			err  error
			resp *WholeResponse
		)

		if offset%(1024*1024*100) == 0 {
			log.Debug("Processing offset ", offset)
		}

		if mode == "write" {
			buf := make([]byte, reqSize, reqSize)
			_, err = RequestWrite(conn, requests, &WholeRequest{
				header: &block.Request{
					Type:   comm.MSG_TYPE_WRITE_REQUEST,
					Offset: offset,
					Length: reqSize,
				},
				data: buf})
		} else {
			resp, err = RequestRead(conn, requests, &WholeRequest{
				header: &block.Request{
					Type:   comm.MSG_TYPE_READ_REQUEST,
					Offset: offset,
					Length: reqSize,
				}})
			if len(resp.data) != int(reqSize) {
				log.Errorln("Wrong data from read")
			}
		}
		if err != nil {
			log.Errorln("Fail to process data from offset ", offset)
		}
	}
}

func GetNewId() int64 {
	return atomic.AddInt64(&idCounter, 1)
}

func RequestWrite(conn *net.TCPConn, requests chan *WholeRequest, request *WholeRequest) (*WholeResponse, error) {
	var response *WholeResponse
	connRequest := request.header
	connRequest.Id = GetNewId()
	respChan := make(chan *WholeResponse)
	idChanMapMutex.Lock()
	idChanMap[connRequest.Id] = respChan
	idChanMapMutex.Unlock()
	requests <- request

	response = <-respChan
	return response, nil
}

func RequestRead(conn *net.TCPConn, requests chan *WholeRequest, request *WholeRequest) (*WholeResponse, error) {
	var response *WholeResponse
	connRequest := request.header
	connRequest.Id = GetNewId()
	respChan := make(chan *WholeResponse)
	idChanMapMutex.Lock()
	idChanMap[connRequest.Id] = respChan
	idChanMapMutex.Unlock()
	requests <- request

	response = <-respChan
	return response, nil
}
