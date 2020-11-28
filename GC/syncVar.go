package GC

import (
	"bytes"
	"encoding/binary"

	cmp "github.com/mortim-portim/GraphEng/Compression"
)

//
//.d8888. db    db d8b   db  .o88b. db    db  .d8b.  d8888b. 
//88'  YP `8b  d8' 888o  88 d8P  Y8 88    88 d8' `8b 88  `8D 
//`8bo.    `8bd8'  88V8o 88 8P      Y8    8P 88ooo88 88oobY' 
//  `Y8b.    88    88 V8o88 8b      `8b  d8' 88~~~88 88`8b   
//db   8D    88    88  V888 Y8b  d8  `8bd8'  88   88 88 `88. 
//`8888Y'    YP    VP   V8P  `Y88P'    YP    YP   YP 88   YD 
//

//Syncronized Variable                                                       
type SyncVar interface {
	IsDirty() bool
	MakeDirty()
	GetData() []byte
	SetData([]byte)
	Type() byte
}

var RegisteredSyncVarTypes map[byte](func()(SyncVar))
func RegisterSyncVar(idx byte, factory func()(SyncVar)) {
	RegisteredSyncVarTypes[idx] = factory
}
func InitSyncVarStandardTypes() {
	RegisteredSyncVarTypes = make(map[byte](func()(SyncVar)))
	RegisteredSyncVarTypes[INT64SYNCED] = func()(SyncVar){return CreateSyncInt64(0)}
	RegisteredSyncVarTypes[FLOAT64SYNCED] = func()(SyncVar){return CreateSyncFloat64(0)}
	RegisteredSyncVarTypes[STRINGSYNCED] = func()(SyncVar){return CreateSyncString("")}
	RegisteredSyncVarTypes[INT16SYNCED] = func()(SyncVar){return CreateSyncInt16(0)}
	RegisteredSyncVarTypes[BOOLSYNCED] = func()(SyncVar){return CreateSyncBool(false)}
	RegisteredSyncVarTypes[BYTESYNCED] = func()(SyncVar){return CreateSyncByte(0)}
	RegisteredSyncVarTypes[BYTECOORDSYNCED] = func()(SyncVar){return CreateSyncByteCoord(0,0)}
}

func GetSyncVarOfType(t byte) SyncVar {
	return RegisteredSyncVarTypes[t]()
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
func (sv *SyncInt64) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncInt64) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncInt64) GetData() []byte {
	sv.dirty = false
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sv.variable)
	return buf.Bytes()
}
func (sv *SyncInt64) SetData(variable []byte) {
	sv.dirty = false
	buf := bytes.NewBuffer(variable)
	binary.Read(buf, binary.LittleEndian, &sv.variable)
}
func (sv *SyncInt64) Type() byte {
	return INT64SYNCED
}
func CreateSyncInt64(variable int64) *SyncInt64 {
	return &SyncInt64{variable, true}
}

// +-+-+-+-+-+-+-+-+-+
// |S|y|n|c|I|n|t|1|6|
// +-+-+-+-+-+-+-+-+-+
type SyncInt16 struct {
	variable int16
	dirty    bool
}
func (sv *SyncInt16) SetInt(i int16) {
	sv.variable = i
	sv.dirty = true
}
func (sv *SyncInt16) GetInt() int16 {
	return sv.variable
}
func (sv *SyncInt16) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncInt16) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncInt16) GetData() []byte {
	sv.dirty = false
	return cmp.Int16ToBytes(sv.variable)
}
func (sv *SyncInt16) SetData(variable []byte) {
	sv.dirty = false
	sv.variable = cmp.BytesToInt16(variable)
}
func (sv *SyncInt16) Type() byte {
	return INT16SYNCED
}
func CreateSyncInt16(variable int16) *SyncInt16 {
	return &SyncInt16{variable, true}
}

// +-+-+-+-+-+-+-+-+
// |S|y|n|c|B|o|o|l|
// +-+-+-+-+-+-+-+-+
type SyncBool struct {
	variable bool
	dirty    bool
}
func (sv *SyncBool) SetBool(i bool) {
	sv.variable = i
	sv.dirty = true
}
func (sv *SyncBool) GetBool() bool {
	return sv.variable
}
func (sv *SyncBool) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncBool) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncBool) GetData() []byte {
	sv.dirty = false
	return []byte{cmp.BoolToByte(sv.variable)}
}
func (sv *SyncBool) SetData(variable []byte) {
	sv.dirty = false
	sv.variable = cmp.ByteToBool(variable[0])
}
func (sv *SyncBool) Type() byte {
	return BOOLSYNCED
}
func CreateSyncBool(variable bool) *SyncBool {
	return &SyncBool{variable, true}
}

// +-+-+-+-+-+-+-+-+
// |S|y|n|c|B|y|t|e|
// +-+-+-+-+-+-+-+-+
type SyncByte struct {
	variable byte
	dirty    bool
}
func (sv *SyncByte) SetByte(i byte) {
	sv.variable = i
	sv.dirty = true
}
func (sv *SyncByte) GetByte() byte {
	return sv.variable
}
func (sv *SyncByte) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncByte) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncByte) GetData() []byte {
	sv.dirty = false
	return []byte{sv.variable}
}
func (sv *SyncByte) SetData(variable []byte) {
	sv.dirty = false
	sv.variable = variable[0]
}
func (sv *SyncByte) Type() byte {
	return BYTESYNCED
}
func CreateSyncByte(variable byte) *SyncByte {
	return &SyncByte{variable, true}
}

// +-+-+-+-+-+-+-+-+-+-+-+-+-+
// |S|y|n|c|B|y|t|e|C|o|o|r|d|
// +-+-+-+-+-+-+-+-+-+-+-+-+-+
type SyncByteCoord struct {
	x,y int8
	dirty    bool
}
func (sv *SyncByteCoord) Move(dx,dy int8) {
	if dx != 0 || dy != 0 {
		sv.x += dx; sv.y += dy
		sv.dirty = true
	}
}
func (sv *SyncByteCoord) Set(x,y int8) {
	if x != sv.x || y != sv.y {
		sv.x = x; sv.y = y
		sv.dirty = true
	}
}
func (sv *SyncByteCoord) Get() (int8, int8) {
	return sv.x, sv.y
}
func (sv *SyncByteCoord) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncByteCoord) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncByteCoord) GetData() []byte {
	sv.dirty = false
	return []byte{byte(int(sv.x)+128), byte(int(sv.y)+128)}
}
func (sv *SyncByteCoord) SetData(data []byte) {
	sv.dirty = false
	sv.x = int8(int(data[0])-128)
	sv.y = int8(int(data[1])-128)
}
func (sv *SyncByteCoord) Type() byte {
	return BYTECOORDSYNCED
}
func CreateSyncByteCoord(x,y int8) *SyncByteCoord {
	return &SyncByteCoord{x,y, true}
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
func (sv *SyncFloat64) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncFloat64) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncFloat64) GetData() []byte {
	sv.dirty = false
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sv.variable)
	return buf.Bytes()
}
func (sv *SyncFloat64) SetData(variable []byte) {
	sv.dirty = false
	buf := bytes.NewBuffer(variable)
	binary.Read(buf, binary.LittleEndian, &sv.variable)
}
func (sv *SyncFloat64) Type() byte {
	return FLOAT64SYNCED
}
func CreateSyncFloat64(variable float64) *SyncFloat64 {
	return &SyncFloat64{variable, true}
}

// +-+-+-+-+-+-+-+-+-+-+
// |S|y|n|c|S|t|r|i|n|g|
// +-+-+-+-+-+-+-+-+-+-+
type SyncString struct {
	variable string
	dirty    bool
}
func (sv *SyncString) Clear() {
	sv.variable = ""
}
func (sv *SyncString) SetBs(bs []byte) {
	sv.variable = string(bs)
	sv.dirty = true
}
func (sv *SyncString) GetBs() []byte {
	return []byte(sv.variable)
}
func (sv *SyncString) SetString(i string) {
	sv.variable = i
	sv.dirty = true
}
func (sv *SyncString) GetString() string {
	return sv.variable
}
func (sv *SyncString) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncString) MakeDirty() {
	sv.dirty = true
}
func (sv *SyncString) GetData() []byte {
	sv.dirty = false
	return []byte(sv.variable)
}
func (sv *SyncString) SetData(variable []byte) {
	sv.dirty = false
	sv.variable = string(variable)
}
func (sv *SyncString) Type() byte {
	return STRINGSYNCED
}
func CreateSyncString(variable string) *SyncString {
	return &SyncString{variable, true}
}

//TODO
/**
// +-+-+-+-+-+-+-+-+-+
// |S|y|n|c|S|t|a|c|k|
// +-+-+-+-+-+-+-+-+-+
type SyncStack struct {
	bytes [][]byte
	dirty    bool
}
func (sv *SyncString) SetString(i string) {
	sv.variable = i
	sv.dirty = true
}
func (sv *SyncString) GetString() string {
	return sv.variable
}
func (sv *SyncString) IsDirty() bool {
	return sv.dirty
}
func (sv *SyncString) GetData() []byte {
	sv.dirty = false
	return []byte(sv.variable)
}
func (sv *SyncString) SetData(variable []byte) {
	sv.dirty = false
	sv.variable = string(variable)
}
func (sv *SyncString) Type() byte {
	return STRINGSYNCED
}
func CreateSyncString(variable string) *SyncString {
	return &SyncString{variable, true}
}
**/