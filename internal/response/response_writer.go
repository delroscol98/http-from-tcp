package response

import (
	"errors"
	"fmt"
	"io"

	"github.com/delroscol98/httpfromtcp/internal/headers"
)

type WriterState int

const (
	WritingStatusLine WriterState = iota
	WritingHeaders
	WritingBody
	WritingTrailers
)

const HTTPVersion = "HTTP/1.1"

type StatusLine struct {
	HttpVersion  string
	StatusCode   StatusCode
	ReasonPhrase string
}

type Writer struct {
	Writer io.Writer
	State  WriterState
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.State != WritingStatusLine {
		return errors.New("Writer state needs to be updated for writing status line")
	}

	var reasonPhrase string
	switch statusCode {
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	case StatusOK:
		reasonPhrase = "OK"
	default:
		return errors.New("unknown status code")
	}

	statusLine := fmt.Appendf(make([]byte, 0), "%v %v %v\r\n", HTTPVersion, statusCode, reasonPhrase)

	_, err := w.Writer.Write(statusLine)
	if err != nil {
		return fmt.Errorf("Error writing status line: %v", err)
	}

	w.State = WritingHeaders
	return nil
}

func (w *Writer) WriteHeaders(h headers.Headers) error {
	if w.State != WritingHeaders {
		return errors.New("Writer state needs to be updated for writing headers")
	}

	for key, value := range h {
		headers := fmt.Appendf(make([]byte, 0), "%s: %s\r\n", key, value)
		_, err := w.Writer.Write(headers)
		if err != nil {
			return fmt.Errorf("Error writing headers: %v", err)
		}
	}

	_, err := w.Writer.Write([]byte("\r\n"))
	w.State = WritingBody
	if err != nil {
		return fmt.Errorf("Error writing headers: %v", err)
	}

	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.State != WritingBody {
		return 0, errors.New("Writer state needs to be updated for writing body")
	}

	n, err := w.Writer.Write(p)
	w.State = WritingStatusLine
	if err != nil {
		return n, fmt.Errorf("Error writing body: %v", err)
	}
	return n, nil
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.State != WritingBody {
		return 0, errors.New("Writer state needs to be updated for writing chunked body")
	}

	n, err := w.Writer.Write(p)
	if err != nil {
		return n, fmt.Errorf("Error writing chunked body: %v", err)
	}
	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() error {
	if w.State != WritingBody {
		return errors.New("Writer state needs to be updated for writing chunked body")
	}

	_, err := w.Writer.Write([]byte("0\r\n"))
	w.State = WritingTrailers
	if err != nil {
		return fmt.Errorf("Error writing end of chunked body: %v", err)
	}

	return nil
}

func (w *Writer) WriteTrailers(t headers.Headers) error {
	if w.State != WritingTrailers {
		return errors.New("Writer state needs to be updated for writing trailers")
	}

	for key, value := range t {
		trailer := fmt.Appendf(make([]byte, 0), "%s: %s\r\n", key, value)
		_, err := w.Writer.Write(trailer)
		if err != nil {
			return fmt.Errorf("Error writing trailers: %v", err)
		}
	}

	_, err := w.Writer.Write([]byte("\r\n"))
	w.State = WritingStatusLine
	if err != nil {
		return fmt.Errorf("Error writing trailers: %v", err)
	}
	return nil
}
