package plumb

import (
	"bytes"
	"crypto/rand"
	"io"
	"log"
	"net"
	"testing"
)

func TestFit(t *testing.T) {
	in, internal1 := net.Pipe()
	internal2, out := net.Pipe()

	payloadSize := 1 << 24
	payload := make([]byte, payloadSize)
	rand.Read(payload)

	// write bytes to the 'in' side of our pipeline
	go func() {
		nbytes, err := io.Copy(in, bytes.NewReader(payload))
		t.Logf("io.Copy(in, in) returned %d, %v", nbytes, err)
	}()

	dest := bytes.NewBuffer(make([]byte, 0, payloadSize))
	go func() {
		// read from our 'out' side of our pipeline
		io.Copy(dest, out)
	}()

	Fit(internal1, internal2)

	if bytes.Compare(dest.Bytes(), payload) != 0 {
		log.Printf("payloads do not match.")
	}
}
