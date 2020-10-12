package GC

import (
	"bytes"
	"encoding/binary"
)

//
//.d8888. db    db d8b   db  .o88b. db    db  .d8b.  d8888b. 
//88'  YP `8b  d8' 888o  88 d8P  Y8 88    88 d8' `8b 88  `8D 
//`8bo.    `8bd8'  88V8o 88 8P      Y8    8P 88ooo88 88oobY' 
//  `Y8b.    88    88 V8o88 8b      `8b  d8' 88~~~88 88`8b   
//db   8D    88    88  V888 Y8b  d8  `8bd8'  88   88 88 `88. 
//`8888Y'    YP    VP   V8P  `Y88P'    YP    YP   YP 88   YD 
//                                                          
type SyncVar interface {
	isDirty() bool
	getData() []byte
	setData([]byte)
	Type() byte
}
func GetSyncVarOfType(t byte) SyncVar {
	switch t {
		case INT64SYNCED:
			return &SyncInt64{}
		case FLOAT64SYNCED:
			return &SyncFloat64{}
		case STRINGSYNCED:
			return &SyncString{}
	}
	return &SyncInt64{}
}

// +-+-+-+-+-+-+-+-+-+
// |S|y|n|c|I|n|t|6|4|
// +-+-+-+-+-+-+-+-+-+
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
func (sv *SyncInt64) Type() byte {
	return INT64SYNCED
}
func CeateSyncInt64(variable int64) *SyncInt64 {
	return &SyncInt64{variable, false}
}

// +-+-+-+-+-+-+-+-+-+-+-+
// |S|y|n|c|F|l|o|a|t|6|4|
// +-+-+-+-+-+-+-+-+-+-+-+
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
func (sv *SyncFloat64) Type() byte {
	return FLOAT64SYNCED
}
func CeateSyncFloat64(variable float64) *SyncFloat64 {
	return &SyncFloat64{variable, false}
}

// +-+-+-+-+-+-+-+-+-+-+
// |S|y|n|c|S|t|r|i|n|g|
// +-+-+-+-+-+-+-+-+-+-+
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
func (sv *SyncString) Type() byte {
	return STRINGSYNCED
}
func CeateSyncString(variable string) *SyncString {
	return &SyncString{variable, false}
}
