package plumb

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/winstonprivacyinc/goproxy/plumb/internal/impatient"
)

func Fit(c1 net.Conn, c2 net.Conn) error {
	return FitBuffer(c1, make([]byte, 32768), c2, make([]byte, 32768))
}

func FitBuffer(c1 net.Conn, buf1 []byte, c2 net.Conn, buf2 []byte) error {
	// create net.Conn pair that updates timeouts
	ic1 := impatient.New(c1, impatient.WithTimeout(2*time.Second))
	ic2 := impatient.New(c2, impatient.WithTimeout(2*time.Second))

	// semaphore to signal io.Copy() completion.
	sem := make(chan struct{})

	go func() {
		io.CopyBuffer(ic2, ic1, buf1)
		sem <- struct{}{}
	}()

	go func() {
		io.CopyBuffer(ic1, ic2, buf2)
		sem <- struct{}{}
	}()

	<-sem
	<-sem

	return nil
}
