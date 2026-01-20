package internal

import (
	"encoding/base32"
	"fmt"
	"net"
	"strings"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// EncodeDataToLabel encodes binary data to a Base32 DNS label (max 63 chars)
func EncodeDataToLabel(data []byte) string {
	return strings.ToLower(b32.EncodeToString(data))
}

// DecodeLabelToData decodes a Base32 DNS label to binary data
func DecodeLabelToData(label string) ([]byte, error) {
	return b32.DecodeString(label)
}

// PackDataToIPv6 packs up to 16 bytes into an IPv6 address
func PackDataToIPv6(data []byte) (net.IP, error) {
	if len(data) > 16 {
		return nil, fmt.Errorf("data too large for IPv6: %d bytes", len(data))
	}
	ip := make([]byte, 16)
	copy(ip, data)
	return net.IP(ip), nil
}

// UnpackIPv6ToData extracts up to 16 bytes from an IPv6 address
func UnpackIPv6ToData(ip net.IP) []byte {
	return []byte(ip.To16())
}
