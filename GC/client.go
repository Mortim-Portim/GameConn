package GC

import (
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/compression"
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
	done, waiting     chan struct{}
	interrupt         chan os.Signal
	InputHandler      func(mt int, msg []byte, err error, c *Client) (alive bool)
	OnCloseConnection func()

	sendMessage, bufferedData []byte
	bufferedWaiting           bool

	confirmed       chan bool
	pendingConfirms int

	sendTimes []time.Time
	Ping      time.Duration

	topLevelLock, lowLevelLock, readLock, confirmLock sync.Mutex
}

func (c *Client) GetPendingConfirms() int {
	return c.pendingConfirms
}
func (c *Client) SendBuffered(bs []byte) {
	printLogF(1, "Sending Buffered data %v\n", bs)
	l := cmp.Int16ToBytes(int16(len(bs)))
	c.bufferedData = append(c.bufferedData, l...)
	c.bufferedData = append(c.bufferedData, bs...)
	printLogF(1, "Pushing Buffered data %v\n", bs)
	c.pushBuffer()
	printLogF(1, "Finished Buffered data %v\n", bs)
}
func (c *Client) pushBuffer() {
	if c.bufferedWaiting {
		return
	}
	c.bufferedWaiting = true

	Data := c.bufferedData
	c.bufferedData = []byte{}
	c.sendSimple(append([]byte{MULTI_MSG}, Data...))
	go func() {
		c.WaitForConfirmation()
		c.bufferedWaiting = false
	}()
}

func (c *Client) SendNormal(bs []byte) {
	c.sendSimple(append([]byte{SINGLE_MSG}, bs...))
}
func (c *Client) sendSimple(bs []byte) {
	printLogF(3, "Sending msg of len(%v) to server\n", len(bs))
	c.topLevelLock.Lock()
	c.pendingConfirms++
	c.sendTimes = append(c.sendTimes, time.Now())
	c.sendMessage = bs
	close(c.waiting)
}
func (c *Client) WaitForConfirmation() {
	c.confirmLock.Lock()
	for c.pendingConfirms > 0 {
		<-c.confirmed
		c.pendingConfirms--
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
		for c != nil {
			c.readLock.Lock()
			mt, msg, err := c.ReadMessage()
			c.readLock.Unlock()
			if err != nil || mt == ws.CloseMessage {
				break
			}
			if mt == ws.BinaryMessage {
				RealMsgType := msg[0]
				msg := msg[1:]
				if RealMsgType == SINGLE_MSG {
					if c.InputHandler != nil && !c.InputHandler(mt, msg, err, c) {
						break
					}
				} else if RealMsgType == MULTI_MSG {
					continuing := false
					for len(msg) > 1 {
						l := int(cmp.BytesToInt16(msg[0:2]))
						data := msg[2 : 2+l]
						msg = msg[l+2:]
						if c.InputHandler != nil {
							continuing = c.InputHandler(mt, data, err, c)
						}
					}
					if !continuing {
						break
					}
				}
				c.lowLevelLock.Lock()
				err2 := c.WriteMessage(ws.PongMessage, []byte{})
				if err2 != nil {
					break
				}
				c.lowLevelLock.Unlock()
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
