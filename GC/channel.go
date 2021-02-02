package GC

import (

)

func GetNewChannel(pipes int) (*Channel) {
	return &Channel{GetBasicSyncVar(), true, make([]int, 0), make([][]byte, pipes)}
}
type Channel struct {
	BasicSyncVar
	dirty    bool
	
	PipeChngs []int
	Pipes [][]byte
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

func (ch *Channel) SendToPipe(idx int, bs []byte) {
	if len(bs) > 0 {
		ch.assignToPipe(idx, bs)
		ch.PipeChngs = append(ch.PipeChngs, idx)
		ch.dirty = true
		ch.ResetUpdated()
	}
}
func (ch *Channel) assignToPipe(idx int, bs []byte) {
	for len(ch.Pipes) < (idx+1) {
		ch.Pipes = append(ch.Pipes, []byte{})
	}
	ch.Pipes[idx] = bs
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