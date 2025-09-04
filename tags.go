package eorm

func shouldEscape(c byte) bool {
	switch c {
	case '\'', '"', '/', '\\', '\n', '\r', '\t', '`', ' ':
		return true
	}
	return false
}

const upperhex = "0123456789ABCDEF"

func NameEncode(s string) string {
	count := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			count++
		}
	}
	if count == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*count
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&0x0f]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}
	return string(t)
}
