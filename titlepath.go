package eorm

import (
	"bytes"
	"strconv"
	"strings"
)

func shouldEscape(c byte) bool {
	switch c {
	case '\'', '"', '/', '\\', '\n', '\r', '\t', '`', ' ':
		return true
	}
	return false
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		panic("invalid hex character")
	}
}

const (
	upperhex  = "0123456789ABCDEF"
	separator = "/"
)

type EscapeError string

func (e EscapeError) Error() string {
	return "invalid title escape " + strconv.Quote(string(e))
}

func TitleEscape(t string) string {
	count := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		if shouldEscape(c) {
			count++
		}
	}
	if count == 0 {
		return t
	}

	var buf [64]byte
	var s []byte

	required := len(t) + 2*count
	if required <= len(buf) {
		s = buf[:required]
	} else {
		s = make([]byte, required)
	}

	j := 0
	for i := 0; i < len(t); i++ {
		c := t[i]
		if shouldEscape(c) {
			s[j] = '%'
			s[j+1] = upperhex[c>>4]
			s[j+2] = upperhex[c&0x0f]
			j += 3
		} else {
			s[j] = c
			j++
		}
	}
	return string(s)
}

func TitleUnescape(t string) (string, error) {
	n := 0
	for i := 0; i < len(t); {
		switch t[i] {
		case '%':
			n++
			if i+2 >= len(t) || !ishex(t[i+1]) || !ishex(t[i+2]) {
				t = t[i:]
				if len(t) > 3 {
					t = t[:3]
				}
				return "", EscapeError(t)
			}
			i += 3
		default:
			i++
		}
	}
	if n == 0 {
		return string(t), nil
	}
	var b bytes.Buffer
	b.Grow(len(t) - 2*n)
	for i := 0; i < len(t); i++ {
		switch t[i] {
		case '%':
			b.WriteByte(unhex(t[i+1])<<4 | unhex(t[i+2]))
			i += 2
		default:
			b.WriteByte(t[i])
		}
	}
	return b.String(), nil
}

type TitlePath []string

func (np TitlePath) Encode() string {
	if len(np) == 0 {
		return ""
	}
	parts := make([]string, 0, len(np))
	for _, name := range np {
		parts = append(parts, TitleEscape(name))
	}
	return strings.Join(parts, separator)
}

func (np TitlePath) Decode(namepath string) TitlePath {
	parts := strings.Split(namepath, separator)
	return TitlePath(parts)
}
