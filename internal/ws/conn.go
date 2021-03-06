package ws

import (
	"io"
	"net"
	"strings"
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

const WriteTimeout = 10 * time.Second

func NewClient(conn *websocket.Conn) *Conn {
	return &Conn{
		conn:   conn,
		buff:   nil,
		closed: atomic.NewBool(false),
	}
}

func (c *Conn) Read(dst []byte) (int, error) {
	c.r.Lock()
	defer c.r.Unlock()

	// use buffer or read new message
	var src []byte
	if len(c.buff) > 0 {
		src = c.buff
		c.buff = nil
	} else if _, msg, err := c.conn.ReadMessage(); err == nil {
		src = msg
	} else {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) || strings.HasSuffix(err.Error(), "use of closed network connection") {
			return 0, io.EOF
		}

		return 0, errors.Wrap(err, "unable to read socket")
	}

	// copy src->dest
	var n int
	if ldst := len(dst); len(src) > ldst {
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
			time.Now().Add(WriteTimeout),
		)
		if err != nil && !errors.Is(err, websocket.ErrCloseSent) {
			err = errors.Wrap(err, "unable to send close control message")
			log.Err(err).Msg("unable to close connection")
			return err
		}

		return errors.Wrap(c.conn.Close(), "unable to close connection")
	}

	return nil
}

func (c *Conn) CloseWrite() error {
	return c.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return errors.Wrap(err, "unable to set read deadline")
	}
	return errors.Wrap(c.conn.SetWriteDeadline(t), "unable to set write deadline")
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return errors.Wrap(c.conn.SetReadDeadline(t), "unable to set read deadline")
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return errors.Wrap(c.conn.SetWriteDeadline(t), "unable to set write deadline")
}

func (c *Conn) SendPing() error {
	err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(WriteTimeout))

	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) || strings.HasSuffix(err.Error(), "use of closed network connection") || errors.Is(err, websocket.ErrCloseSent) {
			return io.EOF
		}

		return errors.Wrap(err, "unable to send ping control messaged")
	}

	return nil
}
