package ws

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
)

type Conn struct {
	conn   *websocket.Conn
	buff   []byte
	r      sync.Mutex
	closed *atomic.Bool
}

var _ net.Conn = (*Conn)(nil)

func NewClient(conn *websocket.Conn) *Conn {
	return &Conn{
		conn:   conn,
		closed: atomic.NewBool(false),
	}
}

func (c *Conn) Read(dst []byte) (int, error) {
	c.r.Lock()
	defer c.r.Unlock()

	ldst := len(dst)
	// use buffer or read new message
	var src []byte
	if len(c.buff) > 0 {
		src = c.buff
		c.buff = nil
	} else if _, msg, err := c.conn.ReadMessage(); err == nil {
		src = msg
	} else {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			return 0, io.EOF
		}
		return 0, errors.Wrap(err, "unable to read socket")
	}

	// copy src->dest
	var n int
	if len(src) > ldst {
		// copy as much as possible of src into dst
		n = copy(dst, src[:ldst])
		// copy remainder into buffer
		r := src[ldst:]
		lr := len(r)
		c.buff = make([]byte, lr)
		copy(c.buff, r)
	} else {
		// copy all of src into dst
		n = copy(dst, src)
	}

	// return bytes copied
	return n, nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	err = c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, errors.Wrap(err, "unable to write data")
	}
	return len(b), nil
}

func (c *Conn) Close() error {
	if c.closed.CAS(false, true) {
		err := c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "encore"),
			time.Now().Add(10*time.Second),
		)
		if err != nil && !errors.Is(err, websocket.ErrCloseSent) {
			err = errors.Wrap(err, "unable to send close control message")
			log.Err(err).Msg("unable to close connection")
			return err

		}

		return c.conn.Close()
	}

	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *Conn) SendPing() error {
	return c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
}
