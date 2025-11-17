package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/delroscol98/httpfromtcp/internal/headers"
)

type parserState int

const (
	parserInitialised parserState = iota
	parserParsingHeaders
	parserParsingBody
	parserDone
)

const bufferSize = 8

type Request struct {
	RequestLine RequestLine
	ParserState parserState
	Headers     headers.Headers
	Body        []byte
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize)
	var readToIndex int
	req := Request{
		ParserState: parserInitialised,
		Headers:     headers.NewHeaders(),
		Body:        make([]byte, 0),
	}

	for req.ParserState != parserDone {
		if readToIndex >= cap(buf) {
			newBuf := make([]byte, cap(buf)*2)
			copy(newBuf, buf)
			buf = newBuf
		}

		numBytesRead, err := reader.Read(buf[readToIndex:])
		if err != nil {
			if err == io.EOF {
				if req.ParserState != parserDone {
					return nil, errors.New("incomplete request")
				}

				val, _ := req.Headers.Get("content-length")
				contentLength, _ := strconv.Atoi(val)
				if len(req.Body) < contentLength {
					return nil, errors.New("length of parsed body is not equal to content-length")
				}
				break
			}

			return nil, err
		}
		readToIndex += numBytesRead
		numBytesConsumed, err := req.parse(buf[:readToIndex])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[numBytesConsumed:])
		readToIndex -= numBytesConsumed
	}

	return &req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	var totalBytesParsed int
	for r.ParserState != parserDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return totalBytesParsed, err
		}

		if n == 0 {
			return totalBytesParsed, nil
		}

		totalBytesParsed += n
	}
	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.ParserState {
	case parserInitialised:
		requestLine, numBytesConsumed, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}

		if numBytesConsumed == 0 {
			return 0, nil
		}

		r.RequestLine = *requestLine
		r.ParserState = parserParsingHeaders

		return numBytesConsumed, nil

	case parserParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.ParserState = parserParsingBody
		}
		return n, nil

	case parserParsingBody:
		value, exists := r.Headers.Get("Content-Length")
		if !exists {
			r.ParserState = parserDone
			return len(data), nil
		}

		contentLength, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("Malformed Content-Length: %s", err)
		}

		if len(data) == 0 && contentLength == 0 {
			r.ParserState = parserDone
			return 0, nil
		}

		r.Body = append(r.Body, data...)

		if len(r.Body) > contentLength {
			return 0, fmt.Errorf("content length (%d) header cannot be greater than body length (%d)", contentLength, len(r.Body))
		}

		if len(r.Body) == contentLength {
			r.ParserState = parserDone
		}

		return len(data), nil

	case parserDone:
		return 0, errors.New("error: trying to read data in a done state")
	default:
		return 0, errors.New("error: unknown state")
	}
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		return nil, 0, nil
	}

	requestLineText := string(data[:idx])
	requestLine, err := requestLineFromString(requestLineText)
	if err != nil {
		return nil, 0, err
	}

	return requestLine, idx + 2, nil
}

func requestLineFromString(str string) (*RequestLine, error) {
	parts := strings.Split(str, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("poorly formatted request-line: %s", str)
	}

	method := parts[0]
	for _, c := range method {
		if c < 'A' || c > 'Z' {
			return nil, fmt.Errorf("invalid method: %s", method)
		}
	}

	requestTarget := parts[1]

	versionParts := strings.Split(parts[2], "/")
	if len(versionParts) != 2 {
		return nil, fmt.Errorf("malformed start-line: %s", str)
	}

	httpPart := versionParts[0]
	if httpPart != "HTTP" {
		return nil, fmt.Errorf("Unrecognised HTTP-version: %s", httpPart)
	}

	httpVersion := versionParts[1]
	if httpVersion != "1.1" {
		return nil, fmt.Errorf("Unrecognised HTTP-version: %s", httpVersion)
	}

	return &RequestLine{
		HttpVersion:   httpVersion,
		RequestTarget: requestTarget,
		Method:        method,
	}, nil
}
