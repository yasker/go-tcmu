package rpc

import (
	"io"
	"net"
	"sync"
)

type RequestHandler func(*Request) (*Response, error)

type Server struct {
	conn           *net.TCPConn
	requests       chan *Request
	responses      chan *Response
	workers        int
	waitGroup      *sync.WaitGroup
	requestHandler RequestHandler
}

func NewServer(c *net.TCPConn, workers int, handler RequestHandler) *Server {
	server := &Server{
		conn:           c,
		responses:      make(chan *Response, workers),
		requests:       make(chan *Request, workers),
		workers:        workers,
		requestHandler: handler,
	}

	server.waitGroup = &sync.WaitGroup{}
	server.waitGroup.Add(workers)
	return server
}

func (s *Server) Start() {
	go s.startResponseProcess()
	go s.startRequestProcess()
	for i := 0; i < s.workers; i++ {
		go func() {
			defer s.waitGroup.Done()
			s.requestWorker()
		}()
	}
}

func (s *Server) Stop() {
	close(s.requests)
	close(s.responses)
	s.waitGroup.Wait()
}

func (s *Server) startResponseProcess() {
	for resp := range s.responses {
		if err := SendResponse(s.conn, resp); err != nil {
			log.Error("Fail to send response: ", err)
			continue
		}
	}
}

func (s *Server) startRequestProcess() {
	for {
		req, err := ReadRequest(s.conn)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error("Fail to read request: ", err)
			continue
		}

		s.requests <- req
	}
}

func (s *Server) requestWorker() {
	for req := range s.requests {
		resp, err := s.requestHandler(req)
		if err != nil {
			log.Error("Error handling request: ", err)
			continue
		}
		s.responses <- resp
	}
}
