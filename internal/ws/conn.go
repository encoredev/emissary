package ws

import (
	"net"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Conn struct {
	conn *websocket.Conn
	io   sync.Mutex
}

var _ net.Conn = (*Conn)(nil)

func NewClient(conn *websocket.Conn) *Conn {
	return &Conn{
		conn: conn,
	}
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// c.io.Lock()
	// defer c.io.Unlock()

	typ, data, err := c.conn.ReadMessage()

	log.Debug().Bytes("data", data).Msg("reading data")

	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Err(err).Msg("unexpected socket close")
		}

		log.Err(err).Msg("unable to read")
		return 0, errors.Wrap(err, "unable to read data")
	}

	if typ != websocket.TextMessage {
		return 0, errors.Newf("unexpected message type: %d", typ)
	}

	return copy(b, data), nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// c.io.Lock()
	// defer c.io.Unlock()

	log.Debug().Bytes("data", b).Msg("writing data")

	err = c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, errors.Wrap(err, "unable to write data")
	}
	return len(b), nil
}

func (c *Conn) Close() error {
	c.io.Lock()
	defer c.io.Unlock()

	err := c.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(10*time.Second))
	if err != nil {
		err = errors.Wrap(err, "unable to send close control message")
		log.Err(err).Msg("unable to close connection")
		return err
	}

	return c.conn.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) SetDeadline(_ time.Time) error {
	return errors.New("ws.Client: deadline not supported")
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
