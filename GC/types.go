package GC

import (
	ws "github.com/gorilla/websocket"
)

const (
	NEWCONNECTION = 				byte(0)
	CLOSECONNECTION = 				byte(1)
	
	SYNCVAR_REGISTRY = 				byte(2)
	SYNCVAR_REGISTRY_CONFIRMATION =	byte(3)
	SYNCVAR_M_REGISTRY = 			byte(4)
	SYNCVAR_M_REGISTRY_CONFIRMATION=byte(5)
	SYNCVAR_UPDATE = 				byte(6)
	SYNCVAR_DELETION =				byte(7)
	
	CONFIRMATION =					byte(8)
	
	BINARYMSG = 					byte(9)
	
	MESSAGE_TYPES = 				byte(10)
)

const (
	INT64SYNCED = 		byte(0)
	FLOAT64SYNCED = 	byte(1)
	STRINGSYNCED = 		byte(2)
	INT16SYNCED = 		byte(3)
	BOOLSYNCED = 		byte(4)
	BYTESYNCED = 		byte(5)
	BYTECOORDSYNCED = 	byte(6)
	UINT16SYNCED = 		byte(7)
	
	SYNCVAR_TYPES = 	byte(8)
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
	return
}