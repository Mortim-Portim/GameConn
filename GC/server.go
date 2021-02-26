package GC

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/compression"
)

/**
TODO is multiple useres connect at the same time
User disappear
**/

const ARTIFICIAL_CLIENT_PING = 0 //time.Millisecond * 60
const ARTIFICIAL_SERVER_PING = 0 //time.Millisecond * 60

type Server struct {
	Closing, AllConnections []*ws.Conn

	topLevelLock map[*ws.Conn]*sync.Mutex
	dataLock     sync.Mutex
	confirmLocks map[*ws.Conn]*sync.Mutex

	Connections         map[*ws.Conn]chan bool
	Data                map[*ws.Conn]([]byte)
	Confirms            map[*ws.Conn]chan bool
	PendingConfirms     map[*ws.Conn]int
	connCounter         int
	pendingConfirmsLock sync.Mutex

	BufferedData                         map[*ws.Conn]([]byte)
	ClientsWaiting                       map[*ws.Conn]bool
	BufferedDataLock, ClientsWaitingLock sync.Mutex

	MsgStack     []*message
	InputWaiting bool
	upgrader     *ws.Upgrader
	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}

func GetNewServer() (s *Server) {
	s = &Server{}
	s.upgrader = &ws.Upgrader{}
	s.Connections = make(map[*ws.Conn]chan bool)
	s.Data = make(map[*ws.Conn]([]byte))
	s.Confirms = make(map[*ws.Conn]chan bool)
	s.PendingConfirms = make(map[*ws.Conn]int)
	s.connCounter = 0
	s.topLevelLock = make(map[*ws.Conn]*sync.Mutex)
	s.confirmLocks = make(map[*ws.Conn]*sync.Mutex)
	s.Closing = make([]*ws.Conn, 0)
	s.AllConnections = make([]*ws.Conn, 0)
	s.BufferedData = make(map[*ws.Conn]([]byte))
	s.ClientsWaiting = make(map[*ws.Conn]bool)
	return
}
func (s *Server) isConnClosed(c *ws.Conn) bool {
	return containsC(s.Closing, c)
}
func (s *Server) SendBuffered(bs []byte, c *ws.Conn) {
	printLogF(1, "Sending Buffered data %v\n", bs)
	l := cmp.Uint32ToBytes(uint32(len(bs)))
	s.BufferedDataLock.Lock()
	s.BufferedData[c] = append(s.BufferedData[c], l...)
	s.BufferedData[c] = append(s.BufferedData[c], bs...)
	s.BufferedDataLock.Unlock()
	printLogF(1, "Pushing Buffered data %v\n", bs)
	s.pushBuffer(c)
	printLogF(1, "Finished Buffered data %v\n", bs)
}
func (s *Server) PushBufferIfFilled(c *ws.Conn) {
	s.BufferedDataLock.Lock()
	l := len(s.BufferedData[c])
	s.BufferedDataLock.Unlock()
	if l > 0 {
		s.pushBuffer(c)
	}
}
func (s *Server) pushBuffer(c *ws.Conn) {
	s.ClientsWaitingLock.Lock()
	waiting := s.ClientsWaiting[c]
	s.ClientsWaitingLock.Unlock()
	if waiting {
		return
	}
	s.ClientsWaitingLock.Lock()
	s.ClientsWaiting[c] = true
	s.ClientsWaitingLock.Unlock()
	s.BufferedDataLock.Lock()
	Data := s.BufferedData[c]
	s.BufferedData[c] = []byte{}
	s.BufferedDataLock.Unlock()
	s.sendSimple(append([]byte{MULTI_MSG}, Data...), c)

	go func() {
		s.WaitForConfirmation(c)
		s.ClientsWaitingLock.Lock()
		s.ClientsWaiting[c] = false
		s.ClientsWaitingLock.Unlock()
	}()
}
func (s *Server) SendNormal(bs []byte, c *ws.Conn) {
	s.sendSimple(append([]byte{SINGLE_MSG}, bs...), c)
}
func (s *Server) sendSimple(bs []byte, c *ws.Conn) {
	if s.isConnClosed(c) {
		return
	}
	s.topLevelLock[c].Lock()
	printLogF(3, "Sending msg of len(%v) to connection %p\n", len(bs), c)
	printLogF(1, "Msg: %v\n", bs)
	s.pendingConfirmsLock.Lock()
	s.PendingConfirms[c]++
	s.pendingConfirmsLock.Unlock()
	s.dataLock.Lock()
	s.Data[c] = bs
	s.dataLock.Unlock()
	ch, ok := s.Connections[c]
	if ok {
		ch <- true
	}
}
func (s *Server) WaitForConfirmation(c *ws.Conn) {
	if s.isConnClosed(c) {
		printLogF(2, "Conn %p is closing, not waiting\n", c)
		return
	}

	lock := s.confirmLocks[c]
	if lock != nil {
		lock.Lock()
	}
	if ch, ok := s.Confirms[c]; ok {
		s.pendingConfirmsLock.Lock()
		for s.PendingConfirms[c] > 0 {
			printLogF(1, "Waiting for %v confirmations on %p\n", s.PendingConfirms[c], c)
			<-ch
			s.PendingConfirms[c]--
		}
		s.pendingConfirmsLock.Unlock()
		printLogF(1, "Waiting finished on %p\n", c)
	}
	if lock != nil {
		lock.Unlock()
	}
}
func (s *Server) WaitForConfirmations(cs ...*ws.Conn) {
	for _, c := range cs {
		s.WaitForConfirmation(c)
	}
}
func (s *Server) WaitForAllConfirmations() {
	for _, c := range s.AllConnections {
		s.WaitForConfirmation(c)
	}
}
func (s *Server) SendToMultiple(bs []byte, cs ...*ws.Conn) {
	for _, c := range cs {
		s.SendBuffered(bs, c)
	}
}
func (s *Server) SendAll(bs []byte) {
	for _, c := range s.AllConnections {
		s.SendBuffered(bs, c)
	}
}

func (s *Server) RunOnPort(port string) string {
	ipAddrS := GetFullIP(port)
	s.Run(ipAddrS)
	return ipAddrS
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

	c.SetPongHandler(func(appData string) error {
		if _, ok := s.Confirms[c]; ok {
			printLogF(1, "Confirming for %p\n", c)
			s.Confirms[c] <- true
		}
		return nil
	})

	s.AllConnections = append(s.AllConnections, c)
	var Locker, lowLevelLock, confirmL sync.Mutex
	s.confirmLocks[c] = &confirmL
	s.topLevelLock[c] = &Locker
	s.ClientsWaitingLock.Lock()
	s.ClientsWaiting[c] = false
	s.ClientsWaitingLock.Unlock()
	s.Connections[c] = make(chan bool)
	s.Confirms[c] = make(chan bool)
	s.PendingConfirms[c] = 0

	s.connCounter++
	go func() {
		for {
			<-s.Connections[c]
			time.Sleep(ARTIFICIAL_SERVER_PING)
			lowLevelLock.Lock()
			s.dataLock.Lock()
			err = c.WriteMessage(ws.BinaryMessage, s.Data[c])
			if err != nil {
				break
			}
			s.dataLock.Unlock()
			lowLevelLock.Unlock()
			s.dataLock.Lock()
			s.Data[c] = nil
			s.dataLock.Unlock()
			s.topLevelLock[c].Unlock()
		}
	}()

	defer c.Close()
	for {
		mt, msg, err := c.ReadMessage()
		printLogF(1, "Received Msg of len(%v): %v\n", len(msg), msg)
		if err != nil {
			printLogF(4, "Error %p disconnects: %v, msg: %v, mt: %v\n", c, err, msg, mt)
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
			RealMsgType := msg[0]
			msg := msg[1:]
			if RealMsgType == SINGLE_MSG {
				s.handleInput(c, mt, msg, err)
			} else if RealMsgType == MULTI_MSG {
				for len(msg) > 1 {
					l := int(cmp.BytesToUint32(msg[0:4]))
					data := msg[4 : 4+l]
					msg = msg[l+4:]
					s.handleInput(c, mt, data, err)
				}
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
func (s *Server) HandleInput() {
	for _, M := range s.MsgStack {
		s.callInputHandler(M.c, M.mt, M.msg, M.err)
	}
	s.MsgStack = make([]*message, 0)
}

type message struct {
	c   *ws.Conn
	mt  int
	msg []byte
	err error
}

func (s *Server) handleInput(c *ws.Conn, mt int, msg []byte, err error) {
	if s.InputWaiting {
		s.MsgStack = append(s.MsgStack, &message{c, mt, msg, err})
	} else {
		s.callInputHandler(c, mt, msg, err)
	}
}
func (s *Server) callInputHandler(c *ws.Conn, mt int, msg []byte, err error) {
	if s.InputHandler != nil {
		printLogF(4, "Receiving, mt: %v, msg: %v, err: %v, c: %p\n", mt, msg, err, c)
		s.InputHandler(c, mt, msg, err, s)
	}
}
func (s *Server) closeConn(c *ws.Conn) {
	s.Closing = append(s.Closing, c)
	printLogF(4, "Closing connection to %p\n", c)
	time.Sleep(time.Millisecond)

	printLogF(1, "Locking top level for %p\n", c)
	s.topLevelLock[c].Lock()
	//s.confirmLocks[c].Lock()
	printLogF(1, "Releasing pending confirms for %p\n", c)
	s.PendingConfirms[c] = 1
	if _, ok := s.Confirms[c]; ok {
		close(s.Confirms[c])
	}

	printLogF(1, "Deleting map entries for %p\n", c)
	delete(s.topLevelLock, c)
	delete(s.confirmLocks, c)
	delete(s.Connections, c)
	s.dataLock.Lock()
	delete(s.Data, c)
	s.dataLock.Unlock()
	delete(s.Confirms, c)
	delete(s.PendingConfirms, c)

	printLogF(1, "Removing %p from Allconnections: %v\n", c, s.AllConnections)
	s.AllConnections = removeC(s.AllConnections, c)
	s.connCounter--
	printLogF(4, "Finished Closing Connection %p: AllConnections: %v, counter: %v\n", c, s.AllConnections, s.connCounter)
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
