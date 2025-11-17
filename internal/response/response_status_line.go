package response

import (
	"fmt"
	"io"
)

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func GetStatusLine(statusCode StatusCode) []byte {
	var reasonPhrase string
	switch statusCode {
	case StatusOK:
		reasonPhrase = "OK"
	case StatusBadRequest:
		reasonPhrase = "Bad Request"
	case StatusInternalServerError:
		reasonPhrase = "Internal Server Error"
	}

	return fmt.Appendf(make([]byte, 0), "HTTP/1.1 %d %s\r\n", statusCode, reasonPhrase)
}

func WriteStatusLine(w io.Writer, statuStatusCode StatusCode) error {
	data := GetStatusLine(statuStatusCode)

	_, err := w.Write(data)
	if err != nil {
		return fmt.Errorf("Error writing status line: %w", err)
	}
	return nil
}
