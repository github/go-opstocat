package opstocat

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"testing"
)

func TestSignedWriterSignsPayloads(t *testing.T) {
	var buf bytes.Buffer
	signedBuf := &StatsdSignedWriter{Writer: &buf, Key: []byte("secret")}

	n, err := signedBuf.Write([]byte("abc"))
	if n != 3 {
		t.Fatalf("Expected 3 bytes to be written, but was %v", n)
	} else if err != nil {
		t.Fatal(err)
	}

	signedBytes := buf.Bytes()
	if len(signedBytes) != 32+8+4+3 {
		// signature (32) + timestamp(8) + nonce(4) + message(3)
		t.Fatalf("Expected 47 bytes to be written to the underlying, but %v were written", len(signedBytes))
	}
	hmacBytes := signedBytes[0:32]
	payload := signedBytes[32:]

	mac := hmac.New(sha256.New, signedBuf.Key)
	mac.Write(payload)
	if bytes.Compare(mac.Sum(nil), hmacBytes) != 0 {
		t.Fatalf("HMAC did not match up")
	}
}
