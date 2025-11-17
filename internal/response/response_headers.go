package response

import (
	"fmt"
	"io"

	"github.com/delroscol98/httpfromtcp/internal/headers"
)

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.SetHeaders("Content-Length", fmt.Sprintf("%d", contentLen))
	h.SetHeaders("Connection", "close")
	h.SetHeaders("Content-Type", "text/plain")

	return h
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for key, val := range headers {
		_, err := fmt.Fprintf(w, "%s: %s\r\n", key, val)
		if err != nil {
			return fmt.Errorf("Error writing headers: %w", err)
		}
	}
	_, err := w.Write([]byte("\r\n"))
	if err != nil {
		return fmt.Errorf("Error writing CRLF: %w", err)
	}
	return nil
}
