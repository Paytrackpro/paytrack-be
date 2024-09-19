package webserver

import (
	"net/http"

	socketio "github.com/googollee/go-socket.io"
)

func NewSocketServer() *socketio.Server {
	server := socketio.NewServer(nil)
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		return nil
	})

	server.OnEvent("/", "join", func(s socketio.Conn, msg string) {
		s.Join(msg)
	})

	server.OnEvent("/", "left", func(s socketio.Conn, msg string) {
		s.Leave(msg)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		s.Close()
	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		s.Close()
	})
	return server
}

func (s *WebServer) handleSocket() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Logintype, Content-Type, Content-Length, X-CSRF-Token, Token, session, Origin, Host, Connection, Accept-Encoding, Accept-Language, X-Requested-With")
		r.Header.Del("Origin")
		s.socket.ServeHTTP(w, r)
	}
}
