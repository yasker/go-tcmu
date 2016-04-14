package main

import (
	"flag"
	"net"
	"os"
	"runtime/pprof"
	"sync"
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

	timeout = 5 // in seconds
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

func processData(conn *net.TCPConn) {
	before := time.Now()
	reqSize := int64(*requestSize)

	client := comm.NewClient(conn, 5, *workers)

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
	bandwidthBits := bandwidth * 8
	iops := float64(sizeInBytes) / float64(reqSize) / float64(seconds)
	log.Debugf("Processing done, speed at %.2f MB/sec(%.2f Mb/sec), %.0f request/seconds",
		bandwidth, bandwidthBits, iops)
}

func process(client *comm.Client, mode string, reqSize int64, co chan int64) {
	for offset := range co {
		var (
			err error
			//resp *comm.Response
		)

		if offset%(1024*1024*100) == 0 {
			log.Debug("Processing offset ", offset)
		}

		if mode == "write" {
			buf := make([]byte, reqSize)
			_, err = client.Call(&comm.Request{
				Header: &block.Request{
					Type:   comm.MSG_TYPE_WRITE_REQUEST,
					Offset: offset,
					Length: reqSize,
				},
				Data: buf})
			if err != nil {
				log.Errorln("Fail to process data from offset ", offset, err)
			}
		} else {
			_, err = client.Call(&comm.Request{
				Header: &block.Request{
					Type:   comm.MSG_TYPE_READ_REQUEST,
					Offset: offset,
					Length: reqSize,
				}})
			if err != nil {
				log.Errorln("Fail to process data from offset ", offset, err)
				continue
			}
		}
	}
}
