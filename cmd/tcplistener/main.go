package main

import (
	"fmt"
	"log"
	"net"

	"github.com/delroscol98/httpfromtcp/internal/request"
)

// const inputFilePath = "messages.txt"

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("A connection has been accepted on address: %v", listener.Addr())

		data, err := request.RequestFromReader(connection)
		if err != nil {
			log.Fatal(err)
		}

		method := data.RequestLine.Method
		target := data.RequestLine.RequestTarget
		version := data.RequestLine.HttpVersion

		fmt.Printf("Request line:\n- Method: %s\n- Target: %s\n- Version: %s\n", method, target, version)

		fmt.Println("Headers:")
		for key, value := range data.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println("Body:")
		fmt.Println(string(data.Body))

		fmt.Println("Connection has been closed")
	}
}
