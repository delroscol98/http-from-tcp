package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/delroscol98/httpfromtcp/internal/headers"
	"github.com/delroscol98/httpfromtcp/internal/request"
	"github.com/delroscol98/httpfromtcp/internal/response"
	"github.com/delroscol98/httpfromtcp/internal/server"
)

const port = 42069

func handler(w *response.Writer, req *request.Request) {
	if req.RequestLine.RequestTarget == "/yourproblem" {
		HandlerYourProblem(w, req)
	} else if req.RequestLine.RequestTarget == "/myproblem" {
		HandlerMyProblem(w, req)
	} else if req.RequestLine.RequestTarget == "/" {
		HandlerRoot(w, req)
	} else if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin") {
		HandlerProxy(w, req)
	}
}

func HandlerYourProblem(w *response.Writer, req *request.Request) {
	err := w.WriteStatusLine(response.StatusBadRequest)
	if err != nil {
		log.Fatal(err)
	}
	body := []byte(`<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`)

	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	err = w.WriteHeaders(h)
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.WriteBody(body)
	if err != nil {
		log.Fatal(err)
	}
}

func HandlerMyProblem(w *response.Writer, req *request.Request) {
	err := w.WriteStatusLine(response.StatusInternalServerError)
	if err != nil {
		log.Fatal(err)
	}
	body := []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`)
	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	err = w.WriteHeaders(h)
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.WriteBody(body)
	if err != nil {
		log.Fatal(err)
	}
}

func HandlerRoot(w *response.Writer, req *request.Request) {
	err := w.WriteStatusLine(response.StatusOK)
	if err != nil {
		log.Fatal(err)
	}

	body := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)

	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	err = w.WriteHeaders(h)
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.WriteBody(body)
	if err != nil {
		log.Fatal(err)
	}
}

func HandlerProxy(w *response.Writer, req *request.Request) {
	val := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")

	err := w.WriteStatusLine(response.StatusOK)
	if err != nil {
		log.Fatal(err)
	}

	h := response.GetDefaultHeaders(0)

	h.Delete("Content-Length")
	h.SetHeaders("Transfer-Encoding", "chunked")
	h.SetHeaders("Trailer", "X-Content-Length")
	h.SetHeaders("Trailer", "X-Content-Sha256")

	err = w.WriteHeaders(h)
	if err != nil {
		log.Fatal(err)
	}

	url := fmt.Sprintf("https://httpbin.org%s", val)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	var bytesRead int
	buffer := make([]byte, 8)
	var body []byte
	for {
		if bytesRead >= cap(buffer) {
			newBuffer := make([]byte, 2*cap(buffer))
			copy(newBuffer, buffer)
			buffer = newBuffer
		}

		n, err := res.Body.Read(buffer[bytesRead:])
		if err != nil {
			if err == io.EOF {
				chunkedLengthLine := fmt.Appendf(make([]byte, 0), "%x\r\n", n)
				chunkedBodyLine := fmt.Appendf(buffer[bytesRead:bytesRead+n], "\r\n")
				body = append(body, buffer[bytesRead:bytesRead+n]...)

				w.WriteChunkedBody(chunkedLengthLine)
				w.WriteChunkedBody(chunkedBodyLine)
				w.WriteChunkedBodyDone()
				break
			}
			log.Fatal(err)
		}

		chunkedLengthLine := fmt.Appendf(make([]byte, 0), "%x\r\n", n)
		chunkedBodyLine := fmt.Appendf(buffer[bytesRead:bytesRead+n], "\r\n")
		body = append(body, buffer[bytesRead:bytesRead+n]...)

		w.WriteChunkedBody(chunkedLengthLine)
		w.WriteChunkedBody(chunkedBodyLine)
		bytesRead += n
	}

	hash := sha256.Sum256(body)

	trailers := headers.NewHeaders()
	trailers["X-Content-Length"] = fmt.Sprintf("%d", len(body))
	trailers["X-Content-Sha256"] = fmt.Sprintf("%x", hash)

	err = w.WriteTrailers(trailers)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
