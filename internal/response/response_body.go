package response

import (
	"io"
)

func WriteBody(w io.Writer, body []byte) error {
	_, err := w.Write(body)
	if err != nil {
		return err
	}
	return nil
}
