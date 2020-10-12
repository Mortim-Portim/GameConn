package GC

import (
	"errors"
	"net/http"
	"time"

	ws "github.com/gorilla/websocket"
)

const NEWCONN = "Open"
const CLOSECONN = "Close"

type Server struct {
	conns       map[int]*ws.Conn
	Connections map[int]chan struct{}
	Data        map[int]([]byte)
	connCounter int

	upgrader     *ws.Upgrader
	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}

func GetNewServer() (s *Server) {
	s = &Server{}
	s.upgrader = &ws.Upgrader{}
	s.conns = make(map[int]*ws.Conn)
	s.Connections = make(map[int]chan struct{})
	s.Data = make(map[int]([]byte))
	s.connCounter = 0
	return
}
func (s *Server) Send(bs []byte, ci int) error {
	s.Data[ci] = bs
	ch, ok := s.Connections[ci]
	if !ok {
		return errors.New("Cannot send to unknown connection")
	}
	close(ch)
	return nil
}
func (s *Server) SendToMultiple(bs []byte, ci ...int) error {
	for _,i := range(ci) {
		err := s.Send(bs, i)
		if err != nil {return err}
	}
	return nil
}

func (s *Server) SendAll(bs []byte) error {
	for i := 0; i < s.connCounter; i++ {
		s.Data[i] = bs
		ch, ok := s.Connections[i]
		if !ok {
			return errors.New("Cannot send to unknown connection")
		}
		close(ch)
	}

	return nil
}

//addr := "localhost:8080"
func (s *Server) Run(addr string) {
	go func() {
		http.HandleFunc("/", s.home)
		http.ListenAndServe(addr, nil)
	}()
	time.Sleep(time.Millisecond)
}
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	idx := s.connCounter
	waiting := make(chan struct{})
	s.Connections[idx] = waiting
	s.connCounter++
	go func() {
		for {
			select {
			case <-s.Connections[idx]:
				err = c.WriteMessage(ws.BinaryMessage, s.Data[idx])
				if err != nil {
					break
				}
				s.Connections[idx] = make(chan struct{})
			}
		}
	}()

	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if string(msg) == NEWCONN {
			if s.OnNewConn != nil {
				s.OnNewConn(c, mt, msg, err, s)
			}
		} else if string(msg) == CLOSECONN {
			if s.OnNewConn != nil {
				s.OnCloseConn(c, mt, msg, err, s)
			}
		} else {
			if s.OnNewConn != nil {
				s.InputHandler(c, mt, msg, err, s)
			}
		}
		if err != nil {
			break
		}
	}
}
