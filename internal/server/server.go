package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/delroscol98/httpfromtcp/internal/request"
	"github.com/delroscol98/httpfromtcp/internal/response"
)

type HandlerError struct {
	StatusCode   response.StatusCode
	ErrorMessage string
}

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	listener net.Listener
	handler  Handler
	Closed   atomic.Bool
}

func (h HandlerError) Error() string {
	return fmt.Sprintf("Error StatusCode: %d\nError Message: %s", h.StatusCode, h.ErrorMessage)
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, errors.New("failed to create listener")
	}
	server := Server{
		listener: listener,
		handler:  handler,
	}

	go server.listen()

	return &server, nil
}

func (s *Server) Close() error {
	s.Closed.Store(true)
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.Closed.Load() {
				return
			}
			log.Printf("error accepting connections: %v", err)
		}

		go func() {
			s.handle(conn)
		}()
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	writer := response.Writer{
		Writer: conn,
		State:  response.WritingStatusLine,
	}
	req, err := request.RequestFromReader(conn)
	if err != nil {
		err := writer.WriteStatusLine(response.StatusBadRequest)
		if err != nil {
			log.Fatal(err)
		}

		body := fmt.Appendf(make([]byte, 0), "Error parsing request: %v", err)
		err = writer.WriteHeaders(response.GetDefaultHeaders(len(body)))
		if err != nil {
			log.Fatal(err)
		}

		_, err = writer.WriteBody(body)
		if err != nil {
			log.Fatal(err)
		}
	}

	s.handler(&writer, req)
}
