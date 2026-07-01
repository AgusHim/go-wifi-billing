package lib

import (
	"bytes"
	"errors"
	"net"
	"testing"
)

func TestEncodeReadLengthRoundTrip(t *testing.T) {
	lengths := []int{0, 1, 127, 128, 16383, 16384, 65535, 2097151, 2097152}

	for _, length := range lengths {
		encoded := encodeLength(length)
		got, err := readLength(bytes.NewReader(encoded))
		if err != nil {
			t.Fatalf("read length %d: %v", length, err)
		}
		if got != length {
			t.Fatalf("length roundtrip = %d, want %d", got, length)
		}
	}
}

func TestClassifyMikrotikError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "auth", err: ErrMikrotikAuth, want: "auth_failed"},
		{name: "timeout", err: &net.DNSError{IsTimeout: true}, want: "timeout"},
		{name: "unreachable", err: errors.New("connection refused"), want: "unreachable"},
		{name: "unsupported", err: errors.New("unsupported resource kind"), want: "unsupported"},
		{name: "trap", err: errors.New("input does not match any value of profile"), want: "trap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyMikrotikError(tt.err); got != tt.want {
				t.Fatalf("classify = %q, want %q", got, tt.want)
			}
		})
	}
}
