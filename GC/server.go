package GC

import (
	"time"
	"errors"
	"net/http"

	ws "github.com/gorilla/websocket"
)

type Server struct {
	ConnToIdx	map[*ws.Conn]int
	Connections map[int]chan struct{}
	Data        map[int]([]byte)
	Confirms	map[int]chan struct{}
	connCounter int

	upgrader     *ws.Upgrader
	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}

func GetNewServer() (s *Server) {
	s = &Server{}
	s.upgrader = &ws.Upgrader{}
	s.ConnToIdx = 	make(map[*ws.Conn]int)
	s.Connections = make(map[int]chan struct{})
	s.Data = 		make(map[int]([]byte))
	s.Confirms = 	make(map[int]chan struct{})
	s.connCounter = 0
	return
}
func (s *Server) Send(bs []byte, ci int) error {
	s.Confirms[ci] = make(chan struct{})
	s.Data[ci] = bs
	ch, ok := s.Connections[ci]
	if !ok {
		return errors.New("Cannot send to unknown connection")
	}
	close(ch)
	return nil
}
func (s *Server) WaitForConfirmation(ci int) {
	<-s.Confirms[ci]
}
func (s *Server) WaitForConfirmations(ci ...int) {
	for _,i := range(ci) {
		s.WaitForConfirmation(i)
	}
}
func (s *Server) WaitForAllConfirmations() {
	for i := 0; i < s.connCounter; i++ {
		s.WaitForConfirmation(i)
	}
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
		err := s.Send(bs, i)
		if err != nil {return err}
	}
	return nil
}

//addr := "localhost:8080"
//Should only be called with a delay
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
	s.ConnToIdx[c] = idx
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
				s.Data[idx] = nil
			}
		}
	}()

	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if msg[0] == NEWCONNECTION {
			if s.OnNewConn != nil {
				s.OnNewConn(c, mt, msg[1:], err, s)
			}
		} else if msg[0] == CLOSECONNECTION {
			if s.OnCloseConn != nil {
				s.OnCloseConn(c, mt, msg[1:], err, s)
			}
			return
		} else if msg[0] == CONFIRMATION {
			close(s.Confirms[s.ConnToIdx[c]])
		} else {
			if s.InputHandler != nil {
				s.InputHandler(c, mt, msg, err, s)
			}
			err2 := c.WriteMessage(ws.BinaryMessage, []byte{CONFIRMATION})
			if err2 != nil {
				break
			}
		}
	}
}
