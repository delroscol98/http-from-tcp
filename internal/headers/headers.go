package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

const (
	CRLF = "\r\n"
)

func NewHeaders() Headers {
	return make(Headers)
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	lineEnd := bytes.Index(data, []byte(CRLF))
	if lineEnd == -1 {
		return 0, false, nil
	}

	if lineEnd == 0 {
		return 2, true, nil
	}

	line := data[:lineEnd]
	colonIdx := bytes.Index(line, []byte(":"))
	if colonIdx == 0 {
		return 0, true, fmt.Errorf("Poorly formatted header: %s", string(data))
	}

	prev := data[colonIdx-1]
	if prev == ' ' || prev == '\t' {
		return 0, false, fmt.Errorf("Poorly formatted header: %s", string(data))
	}

	key := strings.TrimSpace(strings.ToLower(string(line[:colonIdx])))
	if !h.ValidateKey(key) {
		return 0, false, fmt.Errorf("Invalid key: %s", key)
	}
	value := strings.TrimSpace(string(line[colonIdx+1:]))

	h.SetHeaders(key, value)

	return lineEnd + 2, false, nil
}

func (h Headers) ValidateKey(key string) bool {
	specialChars := "!#$%&'*+-.^_`|~"
	for _, char := range key {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			!strings.ContainsRune(specialChars, char) {
			return false
		}
	}
	return true
}

func (h Headers) SetHeaders(key, value string) {
	val, exists := h[strings.ToLower(key)]
	if exists {
		h[strings.ToLower(key)] = fmt.Sprintf("%s, %s", val, value)
	} else {
		h[strings.ToLower(key)] = value
	}
}

func (h Headers) Get(key string) (string, bool) {
	value, exists := h[strings.ToLower(key)]
	if !exists {
		return "", false
	} else {
		return value, true
	}
}

func (h Headers) Override(key, value string) {
	h[strings.ToLower(key)] = value
}

func (h Headers) Delete(key string) {
	delete(h, strings.ToLower(key))
}
