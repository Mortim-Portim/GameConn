package GC

import (
	"fmt"
	ws "github.com/gorilla/websocket"
)

const (
	NEWCONNECTION = 				byte(0)
	CLOSECONNECTION = 				byte(1)
	SYNCVAR_REGISTRY = 				byte(2)
	SYNCVAR_REGISTRY_CONFIRMATION =	byte(3)
	SYNCVAR_UPDATE = 				byte(4)
	SYNCVAR_DELETION =				byte(5)
)

const (
	INT64SYNCED = 		byte(0)
	FLOAT64SYNCED = 	byte(1)
	STRINGSYNCED = 		byte(2)
)

func DeleteInt(fast bool, i int, src ...int) (dst []int) {
	if fast {
		src[i] = src[len(src)-1] // Copy last element to index i.
		src[len(src)-1] = 0   	 // Erase last element (write zero value).
		dst = src[:len(src)-1]   // Truncate slice.
	}else{
		copy(src[i:], src[i+1:]) // Shift a[i+1:] left one index.
		src[len(src)-1] = 0      // Erase last element (write zero value).
		dst = src[:len(src)-1]   // Truncate slice.
	}
	return
}

func DeleteConn(fast bool, cnn *ws.Conn, src ...*ws.Conn) (dst []*ws.Conn) {
	fmt.Println([]*ws.Conn{cnn}, ":", src)
	for i,c := range(src) {
		if c == cnn {
			if fast {
				src[i] = src[len(src)-1] // Copy last element to index i.
				dst = src[:len(src)-1]   // Truncate slice.
			}else{
				copy(src[i:], src[i+1:]) // Shift a[i+1:] left one index.
				dst = src[:len(src)-1]   // Truncate slice.
			}
		}
	}
	fmt.Println(dst)
	return
}