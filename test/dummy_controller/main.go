package main

import (
	"flag"
	"time"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/yasker/longhorn/block"
)

var (
	log     = logrus.WithFields(logrus.Fields{"pkg": "main"})
	address = "localhost:5000"

	size        = flag.Int("size", 1000, "size for read/write, in MB")
	mode        = flag.String("mode", "write", "read or write")
	requestSize = flag.Int("request-size", 4096, "request size of each IO")

	done = make(chan bool, 1)
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	flag.Parse()

	if *mode != "read" && *mode != "write" {
		log.Fatal("Invalid mode type ", *mode)
	}

	log.Infof("Mode %v, size %vMB, request size %v bytes\n", *mode, *size, *requestSize)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Cannot connect to replica, %v", err)
	}
	defer conn.Close()

	client := block.NewTransferClient(conn)

	go processData(client, *mode)

	log.Info("Start processing")
	<-done
	log.Info("Finish processing")
}

func processData(client block.TransferClient, mode string) {
	var (
		offset int64
		err    error
	)
	before := time.Now()

	sizeInBytes := int64(*size * 1024 * 1024)
	reqSize := int64(*requestSize)
	buf := make([]byte, reqSize, reqSize)
	for offset < sizeInBytes-reqSize {
		if offset%(1024*1024*100) == 0 {
			log.Debug("Processing offset ", offset)
		}
		if mode == "write" {
			_, err = client.Write(context.Background(), &block.WriteRequest{
				Offset:  offset,
				Context: buf,
			})
		} else {
			_, err = client.Read(context.Background(), &block.ReadRequest{
				Offset: offset,
				Length: reqSize,
			})
		}
		if err != nil {
			log.Errorln("Fail to process data from offset ", offset)
		}
		offset += reqSize
	}
	seconds := time.Now().Sub(before).Seconds()
	bandwidth := float64(sizeInBytes) / seconds / 1024 / 1024
	log.Debugf("Processing done, speed at %.2f MB/second", bandwidth)
	done <- true
}
