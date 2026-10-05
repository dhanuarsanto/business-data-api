package http

import (
	"errors"
	"strconv"
	"strings"
)

const maxStringLen = 255

var (
	ErrInvalidTerminal  = errors.New("terminal must be non-negative integer")
	ErrInvalidStatus    = errors.New("status must be 0..32767, 'failed' or 'gagal'")
	ErrInvalidBoolParam = errors.New("boolean param must be true/1/false/0")
)

func parseStringFilter(val string) *string {
	v := strings.TrimSpace(val)
	if v == "" || len(v) > maxStringLen {
		return nil
	}
	return &v
}

func parseTerminalFilter(val string) (*int, error) {
	v := strings.TrimSpace(val)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 2147483647 {
		return nil, ErrInvalidTerminal
	}
	return &n, nil
}

func parseStatusFilter(val string) (status *int16, statusMin *int16, err error) {
	v := strings.TrimSpace(val)
	if v == "" {
		return nil, nil, nil
	}
	if v == "failed" || v == "gagal" {
		min40 := int16(40)
		return nil, &min40, nil
	}
	n, err := strconv.ParseInt(v, 10, 16)
	if err != nil || n < 0 || n > 32767 {
		return nil, nil, ErrInvalidStatus
	}
	v16 := int16(n)
	return &v16, nil, nil
}

func parseBoolFilter(val string) (*bool, error) {
	v := strings.TrimSpace(val)
	if v == "" {
		return nil, nil
	}
	switch v {
	case "true", "1":
		b := true
		return &b, nil
	case "false", "0":
		b := false
		return &b, nil
	default:
		return nil, ErrInvalidBoolParam
	}
}
