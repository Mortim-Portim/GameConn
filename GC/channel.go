package GC

import (

)

func GetNewChannel(pipes int) (*Channel) {
	return &Channel{GetBasicSyncVar(), true, make([]bool, pipes), make([]int, 0), make([][]byte, pipes)}
}
type Channel struct {
	BasicSyncVar
	dirty    bool
	
	justChanged []bool
	PipeChngs []int
	Pipes [][]byte
}
func (ch *Channel) GetAllChanged() []bool {
	return ch.justChanged
}
func (ch *Channel) JustChanged(idx int) bool {
	if len(ch.justChanged) < idx+1 {
		return false
	}
	return ch.justChanged[idx]
}
func (ch *Channel) ResetJustChanged(from, to int) {
	for i,_ := range(ch.justChanged) {
		if i >= from && i <= to {
			ch.justChanged[i] = false
		}
	}
}
func (ch *Channel) GetData() []byte {
	data := ch.PipesToBytes()
	ch.UpdatedPP()
	if ch.AllUpdated() {
		ch.dirty = false
		ch.PipeChngs = ch.PipeChngs[:0]
	}
	return data
}
func (ch *Channel) SetData(bs []byte) {
	ch.dirty = false
	ch.PipeChngs = ch.PipeChngs[:0]
	ch.BytesToPipes(bs)
}

func (ch *Channel) SendToPipe(idx int, bs []byte, force bool) bool {
	chnged := ch.assignToPipe(idx, bs)
	if chnged || force {
		ch.PipeChngs = append(ch.PipeChngs, idx)
		ch.dirty = true
		ch.ResetUpdated()
		return true
	}
	return false
}
func (ch *Channel) assignToPipe(idx int, bs []byte) bool {
	for len(ch.Pipes) < (idx+1) {
		ch.Pipes = append(ch.Pipes, []byte{})
	}
	if !testBsEq(bs, ch.Pipes[idx]) {
		ch.Pipes[idx] = bs
		return true
	}
	return false
}
func (ch *Channel) PipesToBytes() (bs []byte) {
	for _,idx := range(ch.PipeChngs) {
		data := ch.Pipes[idx]
		bs = append(bs, byte(idx), byte(len(data)))
		bs = append(bs, data...)
	}
	return
}
func (ch *Channel) BytesToPipes(bs []byte) {
	for len(bs) > 2 {
		idx := int(bs[0]); l := int(bs[1]); data := bs[2:2+l]; bs = bs[2+l:]
		ch.assignToPipe(idx, data)
		for len(ch.justChanged) < (idx+1) {
			ch.justChanged = append(ch.justChanged, false)
		}
		ch.justChanged[idx] = true
	}
}
func (ch *Channel) IsDirty() bool {
	return ch.dirty
}
func (ch *Channel) MakeDirty() {
	ch.dirty = true
}
func (ch *Channel) Type() byte {
	return CHANNELSYNCED
}
func CreateSyncChannel() *Channel {
	return GetNewChannel(0)
}