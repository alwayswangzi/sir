package sir

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	isDigit = regexp.MustCompile(`(?m)\d+`)
	isHex   = regexp.MustCompile(`(?m)[0-9a-fA-F]+`)
)

func (c *Ctx) GetURILastNumber() (int, error) {
	uri := c.URI()
	if uri == "" {
		return -1, errors.New("uri is empty")
	}
	if strings.Contains(uri, "?") {
		uri = uri[:strings.Index(uri, "?")]
	}
	if strings.HasSuffix(uri, "/") {
		uri = uri[:len(uri)-1]
	}
	p := strings.LastIndex(uri, "/")
	if p == -1 {
		return -1, errors.New("invalid uri")
	}
	number := uri[p+1:]
	if isDigit.FindString(number) == "" {
		return 1, nil
	}
	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return -1, err
	}
	return int(i), nil
}

func (c *Ctx) GetURILastHex() (string, error) {
	uri := c.URI()
	if uri == "" {
		return "", errors.New("uri is empty")
	}
	if strings.Contains(uri, "?") {
		uri = uri[strings.Index(uri, "?")+1:]
	}
	if strings.HasSuffix(uri, "/") {
		uri = uri[:len(uri)-1]
	}
	p := strings.LastIndex(uri, "/")
	if p == -1 {
		return "", errors.New("invalid uri")
	}
	hex := uri[p+1:]
	if isHex.FindString(hex) == "" {
		return "", fmt.Errorf("invalid hex:%s", hex)
	}
	return hex, nil
}
