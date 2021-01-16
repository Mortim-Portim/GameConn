package GC

import (
	"net"
	"fmt"
	"log"
	"time"
	"sync"
	"net/http"
	ws "github.com/gorilla/websocket"
)

const ARTIFICIAL_CLIENT_PING = 0//time.Millisecond*30
const ARTIFICIAL_SERVER_PING = 0//time.Millisecond*30

type Server struct {
	ConnToIdx	map[*ws.Conn]int
	topLevelLock map[int]*sync.Mutex
	dataLock sync.Mutex
	
	Connections map[int]chan bool
	Data        map[int]([]byte)
	Confirms	map[int]chan bool
	PendingConfirms map[int]int
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
	s.Connections = make(map[int]chan bool)
	s.Data = 		make(map[int]([]byte))
	s.Confirms = 	make(map[int]chan bool)
	s.PendingConfirms = make(map[int]int)
	s.connCounter = 0
	s.topLevelLock = make(map[int]*sync.Mutex)
	return
}
func (s *Server) Send(bs []byte, ci int) {
	s.topLevelLock[ci].Lock()
	time.Sleep(ARTIFICIAL_SERVER_PING)
	if _, ok := s.Confirms[ci]; !ok {
		s.Confirms[ci] = make(chan bool)
	}
	if _, ok := s.PendingConfirms[ci]; !ok {
		s.PendingConfirms[ci] = 0
	}
	s.PendingConfirms[ci] ++
	s.dataLock.Lock()
	s.Data[ci] = bs
	s.dataLock.Unlock()
	ch, ok := s.Connections[ci]
	if ok {
		ch <- true
	}
}
func (s *Server) WaitForConfirmation(ci int) {
	if ch, ok := s.Confirms[ci]; ok {
		for s.PendingConfirms[ci] > 0 {
			<-ch
			s.PendingConfirms[ci] --
		}
	}
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
func (s *Server) SendToMultiple(bs []byte, ci ...int) {
	for _,i := range(ci) {
		s.Send(bs, i)
	}
}
func (s *Server) SendAll(bs []byte) {
	for i := 0; i < s.connCounter; i++ {
		s.Send(bs, i)
	}
}

func (s *Server) RunOnPort(port string) string {
	ipAddrS := GetFullIP(port)
	s.Run(ipAddrS)
	return ipAddrS
}

//addr := "localhost:8080"
//Should only be called with a delay
func (s *Server) Run(addr string)  {
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
	
	c.SetPongHandler(func(appData string) error {
		if _, ok := s.Confirms[s.ConnToIdx[c]]; ok {
			s.Confirms[s.ConnToIdx[c]] <- true
		}
		return nil
	})
	
	idx := s.connCounter
	var Locker, lowLevelLock sync.Mutex
	s.topLevelLock[idx] = &Locker
	s.Connections[idx] = make(chan bool)
	s.ConnToIdx[c] = idx
	s.connCounter++
	go func() {
		for {
			<-s.Connections[idx]
			lowLevelLock.Lock()
			s.dataLock.Lock()
			err = c.WriteMessage(ws.BinaryMessage, s.Data[idx])
			if err != nil {
				break
			}
			s.dataLock.Unlock()
			lowLevelLock.Unlock()
			s.dataLock.Lock()
			s.Data[idx] = nil
			s.dataLock.Unlock()
			s.topLevelLock[idx].Unlock()
		}
	}()

	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("Error: %v, msg: %v, mt: %v\n", err, msg, mt)
			s.closeConn(c)
			if s.OnCloseConn != nil {
				s.OnCloseConn(c, mt, msg, err, s)
			}
			break
		}
		if msg[0] == NEWCONNECTION {
			if s.OnNewConn != nil {
				s.OnNewConn(c, mt, msg[1:], err, s)
			}
		} else if msg[0] == CLOSECONNECTION {
			s.closeConn(c)
			if s.OnCloseConn != nil {
				s.OnCloseConn(c, mt, msg[1:], err, s)
			}
			return
		} else {
			if s.InputHandler != nil {
				s.InputHandler(c, mt, msg, err, s)
			}
			lowLevelLock.Lock()
			err2 := c.WriteMessage(ws.PongMessage, []byte{})
			if err2 != nil {
				break
			}
			lowLevelLock.Unlock()
		}
	}
}
func (s *Server) closeConn(c *ws.Conn) {
	s.PendingConfirms[s.ConnToIdx[c]] = 1
	if _, ok := s.Confirms[s.ConnToIdx[c]]; ok {
		close(s.Confirms[s.ConnToIdx[c]])
	}
}
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
func GetFullIP(port string) string {
	ip := GetLocalIP()
	return fmt.Sprintf("%s:%s", ip, port)
}