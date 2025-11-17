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

	req, err := request.RequestFromReader(conn)
	if err != nil {
		log.Fatal(err)
	}
	var writer response.Writer
	s.handler(&writer, req)

	statusLine := fmt.Appendf(make([]byte, 0), "%v %v %v\r\n", writer.StatusLine.HttpVersion, writer.StatusLine.StatusCode, writer.StatusLine.ReasonPhrase)

	var headers []byte
	for key, value := range writer.Headers {
		headers = fmt.Appendf(headers, "%s: %s\r\n", key, value)
	}

	CRLF := []byte("\r\n")

	var trailers []byte
	for key, value := range writer.Trailers {
		trailers = fmt.Appendf(trailers, "%s: %s\r\n", key, value)
	}

	response := append(statusLine, headers...)
	response = append(response, CRLF...)
	response = append(response, writer.Body...)
	response = append(response, trailers...)
	response = append(response, CRLF...)

	fmt.Println(string(response))

	conn.Write(response)
}
