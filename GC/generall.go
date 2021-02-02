package GC

import (
	"log"
	ws "github.com/gorilla/websocket"
)

var PRINT_LOG = true

func printLogF(str string, is ...interface{}) {
	if PRINT_LOG {
		log.Printf(str, is...)
	}
}

//Returns true if e is in s
func containsI(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
//Returns true if e is in s
func containsC(s []*ws.Conn, e *ws.Conn) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
func testBsEq(a, b []byte) bool {
    // If one is nil, the other must also be nil.
    if (a == nil) != (b == nil) { 
        return false; 
    }
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}
func copyBs(bs []byte) (bs2 []byte) {
	bs2 = make([]byte, len(bs))
	copy(bs, bs2)
	return
}
func removeC(cs []*ws.Conn, c *ws.Conn) []*ws.Conn {
	idx := -1
	for i,c2 := range(cs) {
		if c2 == c {
			idx = i
			break
		}
	}
	if idx >= 0 {
		cs[idx] = cs[len(cs)-1]
		cs = cs[:len(cs)-1]
	}
	return cs
}