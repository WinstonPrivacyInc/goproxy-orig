package impatient_test

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/winstonprivacyinc/goproxy/plumb/internal/impatient"
)

func TestImpatient(t *testing.T) {
	payloadSize := int64(1 << 24) // 16 megs

	// create a pipe we can use to test end-to-end net.Conn communiction
	c1, c2 := net.Pipe()
	c3, c4 := net.Pipe()

	ic1 := impatient.New(c1, impatient.WithTimeout(2*time.Second),
		impatient.WithName("external1"), impatient.WithDebugFlag(testing.Verbose()))

	ic2 := impatient.New(c2, impatient.WithTimeout(2*time.Second),
		impatient.WithName("winston1"), impatient.WithDebugFlag(testing.Verbose()))

	ic3 := impatient.New(c3, impatient.WithTimeout(2*time.Second),
		impatient.WithName("winston2"), impatient.WithDebugFlag(testing.Verbose()))

	ic4 := impatient.New(c4, impatient.WithTimeout(2*time.Second),
		impatient.WithName("external2"), impatient.WithDebugFlag(testing.Verbose()))

	// modeling external1 -> winston1 -> winston2 -> external2

	sem := make(chan struct{})

	// plumb ic2 reads to ic1 writes
	go func() {
		nbytes, err := io.Copy(ic2, ic3)

		t.Logf("io.Copy(ic2, ic3) returned %d, %v", nbytes, err)

		sem <- struct{}{}

		if nbytes != 0 {
			t.Fatalf("expected io.Copy() to move %d bytes; moved %d instead.", 0, nbytes)
		}
	}()

	// plumb ic1 reads to ic2 writes
	go func() {
		nbytes, err := io.Copy(ic3, ic2)
		t.Logf("io.Copy(ic3, ic2) returned %d, %v", nbytes, err)

		sem <- struct{}{}

		if nbytes != payloadSize {
			t.Fatalf("expected io.Copy() to move %d bytes; moved %d instead.", payloadSize, nbytes)
		}
	}()

	// copy random bytes into one side of our pipeline
	payload := make([]byte, payloadSize)
	rand.Read(payload)
	go func() {
		nbytes, err := io.Copy(ic1, bytes.NewReader(payload))
		t.Logf("io.Copy(ic1, bytes.NewReader(payload)) returned %d, %v", nbytes, err)

	}()

	// read those bytes from our pipeline.
	buf := bytes.NewBuffer(make([]byte, 0, payloadSize))
	go func() {
		nbytes, err := io.Copy(buf, ic4)
		t.Logf("io.Copy(buf, ic4) returned %d, %v", nbytes, err)
	}()

	t.Logf("waiting for pipline to breakdown.")

	// wait for copies in both directions to complete
	// this is how we might plumb or fit two connectons in another package.
	<-sem
	<-sem

	// see if we get what we put in
	if bytes.Compare(buf.Bytes(), payload) != 0 {
		log.Printf("payloads do not match.")
	}
}
