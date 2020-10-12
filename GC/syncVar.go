package GC

import (
	"bytes"
	"encoding/binary"
)

type SyncVar interface {
	isDirty() bool
	getData() []byte
	setData([]byte)
}

type SyncInt64 struct {
	variable int64
	dirty    bool
}

func (sv *SyncInt64) SetInt(i int64) {
	sv.variable = i
	sv.dirty = true
}

func (sv *SyncInt64) GetInt() int64 {
	return sv.variable
}

func (sv *SyncInt64) isDirty() bool {
	return sv.dirty
}

func (sv *SyncInt64) getData() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sv.variable)
	return buf.Bytes()
}

func (sv *SyncInt64) setData(variable []byte) {
	buf := bytes.NewBuffer(variable)
	binary.Read(buf, binary.LittleEndian, &sv.variable)
}

func CeateSyncInt64(variable int64) *SyncInt64 {
	return &SyncInt64{variable, false}
}

type SyncFloat64 struct {
	variable float64
	dirty    bool
}

func (sv *SyncFloat64) SetFloat(i float64) {
	sv.variable = i
	sv.dirty = true
}

func (sv *SyncFloat64) GetFloat() float64 {
	return sv.variable
}

func (sv *SyncFloat64) isDirty() bool {
	return sv.dirty
}

func (sv *SyncFloat64) getData() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sv.variable)
	return buf.Bytes()
}

func (sv *SyncFloat64) setData(variable []byte) {
	buf := bytes.NewBuffer(variable)
	binary.Read(buf, binary.LittleEndian, &sv.variable)
}

func CeateSyncFloat64(variable float64) *SyncFloat64 {
	return &SyncFloat64{variable, false}
}

type SyncString struct {
	variable string
	dirty    bool
}

func (sv *SyncString) SetString(i string) {
	sv.variable = i
	sv.dirty = true
}

func (sv *SyncString) GetString() string {
	return sv.variable
}

func (sv *SyncString) isDirty() bool {
	return sv.dirty
}

func (sv *SyncString) getData() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sv.variable)
	return buf.Bytes()
}

func (sv *SyncString) setData(variable []byte) {
	buf := bytes.NewBuffer(variable)
	binary.Read(buf, binary.LittleEndian, &sv.variable)
}

func CeateSyncString(variable string) *SyncString {
	return &SyncString{variable, false}
}
