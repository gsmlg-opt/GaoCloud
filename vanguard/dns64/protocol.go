package dns64

import (
	"net"
	"strconv"
	"strings"
)

var validPrefixes = []int{32, 40, 48, 56, 64, 96}

const maxPrefixLen = 96

type Dns64Converter struct {
	prefix    net.IP
	suffix    net.IP
	prefixLen int
}

func converterFromString(s string) (*Dns64Converter, error) {
	preAndSuffix := strings.Split(s, " ")
	pre := preAndSuffix[0]
	preAndLen := strings.Split(pre, "/")
	if len(preAndLen) != 2 {
		return nil, ErrInvalidDns64Prefix
	}

	prefix := net.ParseIP(preAndLen[0])
	if prefix == nil {
		return nil, ErrBadDns64Prefix
	}

	prefixLen, err := strconv.Atoi(preAndLen[1])
	if err != nil {
		return nil, ErrInvalidDns64Prefix
	}
	isPrefixLenValid := false
	for _, valid := range validPrefixes {
		if valid == prefixLen {
			isPrefixLenValid = true
			break
		}
	}
	if isPrefixLenValid == false {
		return nil, ErrInvalidDns64Prefix
	}

	var suffix net.IP
	if prefixLen != maxPrefixLen {
		if len(preAndSuffix) != 2 {
			return nil, ErrInvalidDns64Suffix
		}
		suffix = net.ParseIP(preAndSuffix[1])
		if suffix == nil {
			return nil, ErrBadDns64Suffix
		}
	}

	return &Dns64Converter{
		prefix:    prefix,
		suffix:    suffix,
		prefixLen: prefixLen,
	}, nil
}

func (c *Dns64Converter) synthesisAddr(v4 net.IP) net.IP {
	v6 := make(net.IP, net.IPv6len)
	copyedByte := c.prefixLen / 8
	copy(v6, c.prefix[0:copyedByte])
	v4Index := 0
	for copyedByte < 16 && v4Index < 4 {
		if copyedByte != 8 {
			v6[copyedByte] = v4[v4Index]
			v4Index += 1
		}
		copyedByte += 1
	}

	if copyedByte < 16 {
		copy(v6[copyedByte:], c.suffix[copyedByte:])
	}

	return v6
}
