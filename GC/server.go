package GC

import (
	"net/http"

	ws "github.com/gorilla/websocket"
)
const NEWCONN = "Open"
const CLOSECONN = "Close"

type Server struct {
	Connections map[*ws.Conn]chan []byte
	
	upgrader *ws.Upgrader
	InputHandler func(c *ws.Conn, mt int, msg []byte, err error)
	OnNewConn func(c *ws.Conn, mt int, msg []byte, err error)
	OnCloseConn func(c *ws.Conn, mt int, msg []byte, err error)
}


func GetNewServer() (s *Server) {
	s = &Server{}
	s.upgrader = &ws.Upgrader{}
	s.Connections = make(map[*ws.Conn]chan []byte)
	return
}
func (s *Server) Send(bs []byte, c *ws.Conn) {
	s.Connections[c] <- bs
}
//addr := "localhost:8080"
func (s *Server) Run(addr string) error {
	http.HandleFunc("/", s.home)
	err := http.ListenAndServe(addr, nil)
	if err != nil {return err}
	return nil
}
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	
	waiting := make(chan []byte)
	s.Connections[c] = waiting
	go func(){
		for {
			select {
				case bs := <-s.Connections[c]:
					err = c.WriteMessage(ws.BinaryMessage, bs)
					if err != nil {
						break
					}
					s.Connections[c] = make(chan []byte)
			}
		}
	}()
	
	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if string(msg) == NEWCONN {
			if s.OnNewConn != nil {
				s.OnNewConn(c, mt, msg, err)
			}
		}else if mt == ws.CloseMessage {
			if s.OnNewConn != nil {
				s.OnCloseConn(c, mt, msg, err)
			}
		}else{
			if s.OnNewConn != nil {
				s.InputHandler(c, mt, msg, err)
			}
		}
	}
}