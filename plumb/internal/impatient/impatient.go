package impatient

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

// Conn updates the deadline of a net.Conn for every operation lots of debug
// stuff that can be stripped out after testing.
type Conn struct {
	c        net.Conn
	deadline time.Duration
	name     string
	debug    bool
	log      *log.Logger
}

func WithDebugFlag(d bool) func(*Conn) error {
	return func(c *Conn) error {
		c.debug = d
		return nil
	}
}

func WithDebug() func(*Conn) error {
	return func(c *Conn) error {
		c.debug = !c.debug
		return nil
	}
}

func WithName(n string) func(*Conn) error {
	return func(c *Conn) error {
		c.name = n
		return nil
	}
}

func WithTimeout(t time.Duration) func(*Conn) error {
	return func(c *Conn) error {
		c.deadline = t
		return nil
	}
}

func New(c net.Conn, opts ...func(*Conn) error) *Conn {
	rc, _ := NewWithError(c, opts...)
	return rc
}

func NewWithError(c net.Conn, opts ...func(*Conn) error) (*Conn, error) {
	rc := Conn{
		c: c,
	}

	for _, opt := range opts {
		if err := opt(&rc); err != nil {
			return nil, err
		}
	}

	name := func() string {
		if rc.name == "" {
			return "*UNKNOWN* "
		}
		return rc.name + " "
	}

	writer := func() io.Writer {
		if rc.debug {
			return os.Stderr
		}
		return ioutil.Discard
	}

	rc.log = log.New(writer(), "IMPDEBUG: "+name(), log.LstdFlags|log.Lshortfile)
	deadline := time.Now().Add(rc.deadline)

	// initialize deadline.  this is important.
	rc.c.SetDeadline(deadline)

	return &rc, nil
}

func (c *Conn) Write(p []byte) (int, error) {
	// attempt to write to conn.
	nbytes, err := c.c.Write(p)

	if err != nil {
		return nbytes, err
	}

	// update underlying net.Conn write deadline
	c.c.SetDeadline(time.Now().Add(c.deadline))

	c.log.Printf("wrote %d bytes.", nbytes)

	return nbytes, nil
}

func (c *Conn) Read(p []byte) (int, error) {
	// attempt to read from conn.
	nbytes, err := c.c.Read(p)

	if err != nil {
		return nbytes, err
	}

	// update underlying net.Conn read deadline.
	c.c.SetDeadline(time.Now().Add(c.deadline))

	c.log.Printf("read %d bytes.", nbytes)
	return nbytes, nil
}
