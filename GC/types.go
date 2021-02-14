package GC

import (
	ws "github.com/gorilla/websocket"
)

const (
	NEWCONNECTION = 				byte(iota)
	CLOSECONNECTION
	
	SYNCVAR_REGISTRY
	SYNCVAR_REGISTRY_CONFIRMATION
	SYNCVAR_M_REGISTRY
	SYNCVAR_M_REGISTRY_CONFIRMATION
	SYNCVAR_UPDATE
	SYNCVAR_DELETION
	
	CONFIRMATION
	
	BINARYMSG
	
	MESSAGE_TYPES
)

const (
	INT64SYNCED = 		byte(iota)
	FLOAT64SYNCED
	STRINGSYNCED
	INT16SYNCED
	BOOLSYNCED
	BYTESYNCED
	BYTECOORDSYNCED
	UINT16SYNCED
	CHANNELSYNCED
	
	SYNCVAR_TYPES
)

const (
	SINGLE_MSG = byte(iota)
	MULTI_MSG
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