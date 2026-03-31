package lib

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

var ErrMikrotikAuth = errors.New("mikrotik authentication failed")

type MikrotikClient struct {
	conn net.Conn
}

func NewMikrotikClient(host string, port int, useTLS bool, timeout time.Duration) (*MikrotikClient, error) {
	address := net.JoinHostPort(strings.TrimSpace(host), fmt.Sprintf("%d", port))
	dialer := &net.Dialer{Timeout: timeout}

	var conn net.Conn
	var err error
	if useTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
			InsecureSkipVerify: true, // only for phase-2 diagnostics
		})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return nil, err
	}

	_ = conn.SetDeadline(time.Now().Add(timeout))
	return &MikrotikClient{conn: conn}, nil
}

func (c *MikrotikClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *MikrotikClient) Login(username, password string) error {
	if err := c.loginPlain(username, password); err == nil {
		return nil
	}

	reply, err := c.Run("/login")
	if err != nil {
		return err
	}
	if len(reply) == 0 || strings.TrimSpace(reply[0]["ret"]) == "" {
		return ErrMikrotikAuth
	}

	challenge, err := hex.DecodeString(reply[0]["ret"])
	if err != nil {
		return err
	}
	hash := md5.Sum(append([]byte{0x00}, append([]byte(password), challenge...)...))
	response := "00" + hex.EncodeToString(hash[:])

	_, err = c.Run(
		"/login",
		"=name="+username,
		"=response="+response,
	)
	if err != nil {
		return ErrMikrotikAuth
	}
	return nil
}

func (c *MikrotikClient) loginPlain(username, password string) error {
	_, err := c.Run(
		"/login",
		"=name="+username,
		"=password="+password,
	)
	if err != nil {
		return ErrMikrotikAuth
	}
	return nil
}

func (c *MikrotikClient) Run(words ...string) ([]map[string]string, error) {
	if err := c.writeSentence(words...); err != nil {
		return nil, err
	}

	var replies []map[string]string
	for {
		sentence, err := c.readSentence()
		if err != nil {
			return nil, err
		}
		if len(sentence) == 0 {
			continue
		}

		kind := sentence[0]
		fields := map[string]string{}
		for _, word := range sentence[1:] {
			if !strings.HasPrefix(word, "=") {
				continue
			}
			parts := strings.SplitN(word[1:], "=", 2)
			if len(parts) == 2 {
				fields[parts[0]] = parts[1]
			}
		}

		switch kind {
		case "!re", "!done":
			if len(fields) > 0 {
				replies = append(replies, fields)
			}
			if kind == "!done" {
				return replies, nil
			}
		case "!trap":
			message := fields["message"]
			if message == "" {
				message = "router returned trap response"
			}
			if strings.Contains(strings.ToLower(message), "cannot log in") {
				return nil, ErrMikrotikAuth
			}
			return nil, errors.New(message)
		case "!fatal":
			message := fields["message"]
			if message == "" {
				message = "router returned fatal response"
			}
			return nil, errors.New(message)
		}
	}
}

func (c *MikrotikClient) writeSentence(words ...string) error {
	for _, word := range words {
		if err := c.writeWord(word); err != nil {
			return err
		}
	}
	return c.writeWord("")
}

func (c *MikrotikClient) writeWord(word string) error {
	payload := []byte(word)
	if _, err := c.conn.Write(encodeLength(len(payload))); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := c.conn.Write(payload)
	return err
}

func (c *MikrotikClient) readSentence() ([]string, error) {
	var words []string
	for {
		word, err := c.readWord()
		if err != nil {
			return nil, err
		}
		if word == "" {
			return words, nil
		}
		words = append(words, word)
	}
}

func (c *MikrotikClient) readWord() (string, error) {
	length, err := readLength(c.conn)
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func encodeLength(length int) []byte {
	switch {
	case length < 0x80:
		return []byte{byte(length)}
	case length < 0x4000:
		length |= 0x8000
		return []byte{byte(length >> 8), byte(length)}
	case length < 0x200000:
		length |= 0xC00000
		return []byte{byte(length >> 16), byte(length >> 8), byte(length)}
	case length < 0x10000000:
		length |= 0xE0000000
		return []byte{byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}
	default:
		return []byte{0xF0, byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}
	}
}

func readLength(r io.Reader) (int, error) {
	first := make([]byte, 1)
	if _, err := io.ReadFull(r, first); err != nil {
		return 0, err
	}

	switch {
	case first[0]&0x80 == 0x00:
		return int(first[0]), nil
	case first[0]&0xC0 == 0x80:
		next := make([]byte, 1)
		if _, err := io.ReadFull(r, next); err != nil {
			return 0, err
		}
		return int(first[0]&^0xC0)<<8 | int(next[0]), nil
	case first[0]&0xE0 == 0xC0:
		next := make([]byte, 2)
		if _, err := io.ReadFull(r, next); err != nil {
			return 0, err
		}
		return int(first[0]&^0xE0)<<16 | int(next[0])<<8 | int(next[1]), nil
	case first[0]&0xF0 == 0xE0:
		next := make([]byte, 3)
		if _, err := io.ReadFull(r, next); err != nil {
			return 0, err
		}
		return int(first[0]&^0xF0)<<24 | int(next[0])<<16 | int(next[1])<<8 | int(next[2]), nil
	default:
		next := make([]byte, 4)
		if _, err := io.ReadFull(r, next); err != nil {
			return 0, err
		}
		return int(next[0])<<24 | int(next[1])<<16 | int(next[2])<<8 | int(next[3]), nil
	}
}
