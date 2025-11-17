package response

import (
	"errors"

	"github.com/delroscol98/httpfromtcp/internal/headers"
)

type WriterState int

const (
	WritingStatusLine WriterState = iota
	WritingHeaders
	WritingBody
	WritingChunkedBody
	WritingTrailers
	WritingDone
)

type StatusLine struct {
	HttpVersion  string
	StatusCode   StatusCode
	ReasonPhrase string
}

type Writer struct {
	StatusLine StatusLine
	Headers    headers.Headers
	Body       []byte
	Trailers   headers.Headers
	State      WriterState
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.State != WritingStatusLine {
		return errors.New("Writer state needs to be updated for writing status line")
	}
	w.StatusLine.HttpVersion = "HTTP/1.1"
	w.StatusLine.StatusCode = statusCode

	switch statusCode {
	case StatusBadRequest:
		w.StatusLine.ReasonPhrase = "Bad Request"
	case StatusInternalServerError:
		w.StatusLine.ReasonPhrase = "Internal Server Error"
	case StatusOK:
		w.StatusLine.ReasonPhrase = "OK"
	default:
		return errors.New("unknown status code")
	}

	w.State = WritingHeaders
	return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.State != WritingHeaders {
		return errors.New("Writer state needs to be updated for writing headers")
	}
	w.Headers = headers

	_, exists := w.Headers.Get("Transfer-Encoding")
	if !exists {
		w.State = WritingBody
		return nil
	}

	w.State = WritingChunkedBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.State != WritingBody {
		return 0, errors.New("Writer state needs to be updated for writing body")
	}

	w.Body = p

	w.State = WritingDone
	return len(p), nil
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.State != WritingChunkedBody {
		return 0, errors.New("Writer state needs to be updated for writing chunked body")
	}

	w.Body = append(w.Body, p...)
	return len(p), nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	w.State = WritingTrailers
	return 0, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.State != WritingTrailers {
		return errors.New("Writer state needs to be updated for writing trailers")
	}

	w.Trailers = h
	w.State = WritingDone
	return nil
}
