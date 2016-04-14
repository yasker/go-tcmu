package rpc

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/yasker/longhorn/block"
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "dummy_controller"})
)

type Request struct {
	Header *block.Request
	Data   []byte
}

type Response struct {
	Header *block.Response
	Data   []byte
}

type Client struct {
	conn                *net.TCPConn
	seqRespChanMap      map[int64]chan *Response
	seqRespChanMapMutex *sync.Mutex
	seqCounter          int64
	requests            chan *Request
	timeout             int
}

func NewClient(c *net.TCPConn, timeout, bufSize int) *Client {
	client := &Client{
		conn:                c,
		seqRespChanMap:      make(map[int64]chan *Response),
		seqRespChanMapMutex: &sync.Mutex{},
		seqCounter:          0,
		requests:            make(chan *Request, bufSize),
		timeout:             timeout,
	}

	go client.startRequestProcess()
	go client.startResponseProcess()
	return client
}

func (c *Client) Close() {
	close(c.requests)
}

func (c *Client) startRequestProcess() {
	for req := range c.requests {
		header := req.Header
		if err := SendRequest(c.conn, header); err != nil {
			log.Error("Fail to send request:", err)
			continue
		}
		if header.Type == MSG_TYPE_WRITE_REQUEST {
			if err := SendData(c.conn, req.Data); err != nil {
				log.Error("Fail to send data:", err)
				continue
			}
		}
	}
}

func (c *Client) startResponseProcess() {
	for {
		var (
			response *Response
			data     []byte
		)
		respHeader, err := ReadResponse(c.conn)
		if err != nil {
			if err == io.EOF {
				log.Info("Connection closed")
				break
			}
			log.Error("Fail to read response:", err)
			continue
		}
		if respHeader.Result != "Success" {
			log.Error("Operation failed: ", respHeader.Result)
			continue
		}
		if respHeader.Type == MSG_TYPE_READ_RESPONSE {
			data = make([]byte, respHeader.Length, respHeader.Length)
			if err := ReceiveData(c.conn, data); err != nil {
				log.Error("Receive data failed:", err)
				continue
			}
		}

		c.seqRespChanMapMutex.Lock()
		respChan := c.seqRespChanMap[respHeader.Id]
		delete(c.seqRespChanMap, respHeader.Id)
		c.seqRespChanMapMutex.Unlock()

		response = &Response{
			Header: respHeader,
			Data:   data,
		}
		respChan <- response
	}
}

func (c *Client) GetNewId() int64 {
	return atomic.AddInt64(&c.seqCounter, 1)
}

func (c *Client) Call(request *Request) (*Response, error) {
	var (
		response *Response
		err      error
	)
	request.Header.Id = c.GetNewId()
	respChan := make(chan *Response)
	c.seqRespChanMapMutex.Lock()
	c.seqRespChanMap[request.Header.Id] = respChan
	c.seqRespChanMapMutex.Unlock()
	c.requests <- request

	select {
	case response = <-respChan:
		err = nil
	case <-time.After(time.Duration(c.timeout) * time.Second):
		err = fmt.Errorf("Timeout for operation %v", request.Header.Id)
	}
	return response, err
}
