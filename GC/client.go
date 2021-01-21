package GC

import (
	"net/url"
	"os"
	"os/signal"
	"time"
	"sync"
	ws "github.com/gorilla/websocket"
)

func GetNewClient() (cl *Client) {
	cl = &Client{}
	cl.reset()
	return
}
func (c *Client) reset() {
	c.confirmed = make(chan bool)
	c.sendTimes = make([]time.Time, 0)
	c.Ping = time.Duration(0)
	c.sendMessage = make([]byte, 0)
	c.pendingConfirms = 0
}
type Client struct {
	ws.Conn
	done, waiting chan struct{}
	interrupt     chan os.Signal
	InputHandler  func(mt int, msg []byte, err error, c *Client) (alive bool)
	OnCloseConnection func()
	sendMessage   []byte
	confirmed	  chan bool
	pendingConfirms int
	sendTimes []time.Time
	Ping time.Duration
	
	topLevelLock, lowLevelLock, readLock, confirmLock sync.Mutex
}
func (c *Client) GetPendingConfirms() int {
	return c.pendingConfirms
}
func (c *Client) Send(bs []byte) {
	c.topLevelLock.Lock()
	c.pendingConfirms ++
	c.sendTimes = append(c.sendTimes, time.Now())
	c.sendMessage = bs
	close(c.waiting)
}
func (c *Client) WaitForConfirmation() {
	c.confirmLock.Lock()
	for c.pendingConfirms > 0 {
		<-c.confirmed
		c.pendingConfirms --
		if len(c.sendTimes) > 0 {
			c.Ping = time.Now().Sub(c.sendTimes[0])
			c.sendTimes = c.sendTimes[1:]
		}
	}
	c.confirmLock.Unlock()
	return
}

//addr := "localhost:8080"
func (c *Client) MakeConn(addr string) error {
	//If the system exits c.interrupt fires
	c.interrupt = make(chan os.Signal, 1)
	signal.Notify(c.interrupt, os.Interrupt)

	//Create an URL to dial
	u := url.URL{Scheme: "ws", Host: addr, Path: "/echo"}

	//dial URL and set client connection
	c_tmp, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	c.Conn = *c_tmp
	
	c.Conn.SetPongHandler(func(appData string) error {
		if c.confirmed != nil {
			c.confirmed <- true
		}
		
		return nil
	})

	//receive input in a separate thread
	c.done = make(chan struct{})
	go func() {
		for {
			if c != nil {
				c.readLock.Lock()
				mt, msg, err := c.ReadMessage()
				c.readLock.Unlock()
				if err != nil || mt == ws.CloseMessage {break}
				if mt == ws.BinaryMessage {
					if c.InputHandler != nil && !c.InputHandler(mt, msg, err, c) {break}
					c.lowLevelLock.Lock()
					err2 := c.WriteMessage(ws.PongMessage, []byte{})
					if err2 != nil {
						break
					}
					c.lowLevelLock.Unlock()
				}
			}
		}
		close(c.done)
	}()

	//Send an initial message
	c.lowLevelLock.Lock()
	err = c.WriteMessage(ws.TextMessage, []byte{NEWCONNECTION})
	if err != nil {
		return err
	}
	c.lowLevelLock.Unlock()

	//wait for input an send it on a separate thread
	c.waiting = make(chan struct{})
	go func() {
		for {
			select {
			case <-c.done:
				c.CloseConn()
				return
			case <-c.waiting:
				if c.sendMessage != nil {
					time.Sleep(ARTIFICIAL_CLIENT_PING)
					c.lowLevelLock.Lock()
					err := c.WriteMessage(ws.BinaryMessage, c.sendMessage)
					if err != nil {
						c.CloseConn()
						return
					}
					c.lowLevelLock.Unlock()
					c.sendMessage = nil
				}
				c.waiting = make(chan struct{})
				c.topLevelLock.Unlock()
			case <-c.interrupt:
				c.CloseConn()
				return
			}
		}
	}()
	time.Sleep(time.Millisecond)
	return nil
}
//Should only be called with a delay
func (c *Client) CloseConn() error {
	c.readLock.Lock()
	c.lowLevelLock.Lock()
	c.topLevelLock.Lock()
	
	c.WriteMessage(ws.BinaryMessage, []byte{CLOSECONNECTION})
	if c.OnCloseConnection != nil {
		c.OnCloseConnection()
	}
	time.Sleep(time.Second)
	c.pendingConfirms = 1
	if c.confirmed != nil {
		close(c.confirmed)
	}
	c.reset()
	
	c.readLock.Unlock()
	c.lowLevelLock.Unlock()
	c.topLevelLock.Unlock()
	return nil
}