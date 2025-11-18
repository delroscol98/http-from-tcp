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
	val := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin/")
	url := fmt.Sprintf("https://httpbin.org/%s", val)

	res, err := http.Get(url)
	if err != nil {
		HandlerMyProblem(w, req)
		return
	}
	defer res.Body.Close()

	err = w.WriteStatusLine(response.StatusOK)
	if err != nil {
		log.Fatal(err)
	}

	h := response.GetDefaultHeaders(0)
	h.Delete("Content-Length")
	h.Override("Transfer-Encoding", "chunked")
	h.Override("Trailer", "X-Content-Sha256, X-Content-Length")

	err = w.WriteHeaders(h)
	if err != nil {
		HandlerMyProblem(w, req)
		return
	}

	var body []byte
	for {
		buffer := make([]byte, 1024)
		n, err := res.Body.Read(buffer)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		if n > 0 {
			chunkedLengthLine := fmt.Appendf(make([]byte, 0), "%x\r\n", n)
			chunkedBodyLine := fmt.Appendf(buffer[:n], "\r\n")
			body = append(body, buffer[:n]...)

			_, err := w.WriteChunkedBody(chunkedLengthLine)
			if err != nil {
				fmt.Printf("Error writing chunked body: %v\n", err)
			}

			_, err = w.WriteChunkedBody(chunkedBodyLine)
			if err != nil {
				fmt.Printf("Error writing chunked body: %v\n", err)
			}
		}
	}

	err = w.WriteChunkedBodyDone()
	if err != nil {
		fmt.Printf("Error finishing chunked body: %v\n", err)
	}

	fmt.Print(string(body))

	trailers := headers.NewHeaders()
	trailers.Override("X-Content-Sha256", fmt.Sprintf("%x", sha256.Sum256(body)))
	trailers.Override("X-Content-Length", fmt.Sprintf("%d", len(body)))

	err = w.WriteTrailers(trailers)
	if err != nil {
		fmt.Printf("Error writing trailers: %v", err)
	}
	fmt.Println("Trailers written")
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
