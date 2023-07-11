package httpparser

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

/*
GET
POST
PUT
PATCH
DELETE
OPTIONS
*/

var (
	ErrNoValidMethodFound = errors.New("no valid method found")
	ErrInvalidHeaders     = errors.New("invalid headers")
)

func getMethod(input string) (string, error) {
	for _, m := range []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	} {
		if input[:len(m)] == m {
			return m, nil
		}
	}

	return "", ErrNoValidMethodFound

}

func getURL(input string) (string, error) {
	space := strings.Index(input, " ")
	if space == -1 {
		return input, nil
	}

	return input[:space], nil
}
func getVersion(input string) string {
	newLine := strings.Index(input, "\n")
	if newLine == -1 {
		return ""
	}
	return input[:newLine]
}

func getHeaders(input string) (http.Header, error) {
	output := http.Header{}
	if idx := strings.Index(input, "\n\n"); idx != -1 {
		input = input[:idx]
	}
	headers := strings.Split(input, "\n")
	for _, header := range headers {
		if header == "" {
			break
		}
		kv := strings.Split(header, ":")
		if len(kv) != 2 {
			return nil, ErrInvalidHeaders
		}
		output.Add(kv[0], kv[1])
	}

	return output, nil
}
func Parse(input string) (*http.Request, error) {
	method, err := getMethod(input)
	if err != nil {
		return nil, err
	}

	input = input[len(method)+1:]

	url, err := getURL(input)
	if err != nil {
		return nil, err
	}

	input = input[len(url)+1:]

	version := getVersion(input)

	input = input[len(version)+1:]

	headers, err := getHeaders(input)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if idx := strings.Index(input, "\n\n"); idx != -1 {
		input = input[idx:]
		body = strings.NewReader(input)
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = headers

	return req, nil
}
