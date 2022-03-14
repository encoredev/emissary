package http

import (
	golog "log"
	// "net/http"

	"github.com/armon/go-socks5"
	// "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"
)

func init() {
	router.Methods("GET", "POST").PathPrefix("/proxy").Handler(wsServer)
}

var wsServer = &websocket.Server{
	Config:    websocket.Config{},
	Handshake: nil, // No Handshake
	Handler:   handleProxy,
}

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  409600,
// 	WriteBufferSize: 409600,
// 	CheckOrigin: func(_ *http.Request) bool {
// 		return true
// 	},
// } // use default options

// func handleProxy(w http.ResponseWriter, r *http.Request) {
func handleProxy(ws *websocket.Conn) {
	// var _ net.Conn = ws
	// c, err := upgrader.Upgrade(w, r, nil)
	// if err != nil {
	// 	log.Err(err).Msg("error upgrading websocket")
	// 	return
	// }

	// <-- please auth
	// --> my token
	// <-- auth ok
	// --> connect 127.0.0.1 : 1234
	// <-- connected ok
	//
	// raw := net.Dial("127.0.0.1:1234")

	// conn := ws.NewClient(c)
	// defer conn.Close()
	//
	// for {
	// 	mt, message, err := c.ReadMessage()
	// 	if err != nil {
	// 		log.Printf("read: %+v", err)
	// 		break
	// 	}
	// 	log.Printf("recv: %s", message)
	// 	err = c.WriteMessage(mt, message)
	// 	if err != nil {
	// 		log.Printf("write: %+v", err)
	// 		break
	// 	}
	// }

	// for {
	// 	b := make([]byte, 0, 1024)
	//
	// 	_, err := conn.Read(b)
	// 	if err != nil {
	// 		log.Err(err).Msg("error reading from socket")
	// 		return
	// 	}
	//
	// 	_, err = conn.Write([]byte("hi: " + string(b)))
	// 	if err != nil {
	// 		log.Err(err).Msg("error writing to socket")
	// 		return
	// 	}
	// }

	server, err := socks5.New(&socks5.Config{
		AuthMethods: nil,
		Credentials: socks5.StaticCredentials{"encore": "erocne"},
		Resolver:    nil,
		Rules:       nil,
		Rewriter:    nil,
		BindIP:      nil,
		Logger:      golog.New(log.Logger, "ws", golog.LstdFlags),
		Dial:        nil,
	})
	if err != nil {
		log.Err(err).Msg("unable to start socks5 proxy")
	}

	if err := server.ServeConn(ws); err != nil {
		log.Err(err).Msg("unable to serve socks 5 proxy")
	}
}
