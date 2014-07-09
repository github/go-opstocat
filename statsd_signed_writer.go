package opstocat

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"time"
)

// `StatsdSignedWriter` wraps an `io.Writer`, prefixing writes with an HMAC,
// nonce and timestamp as described in
// <https://github.com/github/statsd-ruby/pull/9>.
//
// As an example:
//     conn, err := net.DialTimeout("udp", "endpoint.example.com:8126", 2*time.Second)
//     if err != nil {
//       // handle err
//     }
//
//     writer := &opstocat.StatsdSignedWriter{Writer: conn, Key: []byte("supersecret")}
//     statter := g2s.New(writer)
//     statter.Counter(1.0, "foo", 1) // :banana: out
type StatsdSignedWriter struct {
	io.Writer
	Key []byte
}

func (s *StatsdSignedWriter) Write(p []byte) (int, error) {
	payload, err := s.signedPayload(p)
	if err != nil {
		return 0, err
	}

	_, err = s.Writer.Write(payload)
	if err != nil {
		return 0, err
	} else {
		return len(p), err
	}
}

func (s *StatsdSignedWriter) signedPayload(p []byte) ([]byte, error) {
	payload := new(bytes.Buffer)

	ts := time.Now()
	binary.Write(payload, binary.LittleEndian, ts.Unix())

	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}
	payload.Write(randomBytes)
	payload.Write(p)

	payloadBytes := payload.Bytes()
	mac := hmac.New(sha1.New, s.Key)
	mac.Write(payloadBytes)

	fullMessage := mac.Sum(nil)
	fullMessage = append(fullMessage, payloadBytes...)
	return fullMessage, nil
}

