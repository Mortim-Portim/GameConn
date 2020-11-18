package GC

import (
	"log"

	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/Compression"
)

func GetNewClientHandler(s *Server, cn *ws.Conn) (ch *ClientHandler) {
	ch = &ClientHandler{Server: s, Conn: cn}
	ch.idCounter = 0
	ch.SyncvarsByID = make(map[int]SyncVar)
	ch.SyncvarsByName = make(map[string]SyncVar)
	ch.NameToID = make(map[string]int)
	return
}

type ClientHandler struct {
	Server         *Server
	Conn           *ws.Conn
	SyncvarsByID   map[int]SyncVar
	SyncvarsByName map[string]SyncVar
	NameToID       map[string]int
	idCounter      int
}

func (ch *ClientHandler) RegisterSyncVar(sv SyncVar, name string) {
	log.Printf("#####Server: Creating SyncVar with ID=%v, name='%s', type=%v , initiated by self=%s", ch.idCounter, name, sv.Type(), ch.Conn.LocalAddr().String())
	ch.SyncvarsByName[name] = sv
	ch.SyncvarsByID[ch.idCounter] = sv
	ch.NameToID[name] = ch.idCounter
	resp := append(append([]byte{SYNCVAR_REGISTRY, sv.Type(), byte(len(name))}, []byte(name)...), cmp.Int16ToBytes(int16(ch.idCounter))...)
	ch.Server.Send(resp, ch.Server.ConnToIdx[ch.Conn])
	ch.idCounter++
}
func (ch *ClientHandler) UpdateSyncVars() {
	var_data := []byte{SYNCVAR_UPDATE}
	for id, sv := range ch.SyncvarsByID {
		if sv.IsDirty() {
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
			log.Printf("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), ch.Conn.LocalAddr().String())
		}
	}
	if len(var_data) > 1 {
		ch.Server.Send(var_data, ch.Server.ConnToIdx[ch.Conn])
	}
}
func (ch *ClientHandler) DeleteSyncVar(name string) {
	id := ch.NameToID[name]
	log.Printf("#####Server: Deleting SyncVar with ID=%v, name='%s', initiated by self=%s", id, name, ch.Conn.LocalAddr().String())
	ch.deleteSyncVarLocal(name, id)
	ch.deleteSyncVarRemote(name, id)
}
func (ch *ClientHandler) deleteSyncVarLocal(name string, id int) {
	delete(ch.SyncvarsByID, id)
	delete(ch.SyncvarsByName, name)
	delete(ch.NameToID, name)
}
func (ch *ClientHandler) deleteSyncVarRemote(name string, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), []byte(name)...)
	ch.Server.Send(append([]byte{SYNCVAR_DELETION}, data...), ch.Server.ConnToIdx[ch.Conn])
}
func (ch *ClientHandler) onSyncVarUpdateC(data []byte) {
	for true {
		id := cmp.BytesToInt16(data[:2])
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4 : 4+l]
		data = data[4+l:]
		ch.SyncvarsByID[int(id)].SetData(dat)
		if len(data) <= 0 {
			break
		}
		log.Printf("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by client=%s", id, l, ch.Conn.RemoteAddr().String())
	}
}
func (ch *ClientHandler) processSyncVarRegistry(data []byte) {
	t := data[0]
	name := string(data[1:])
	id := ch.idCounter
	ch.SyncvarsByID[id] = GetSyncVarOfType(t)
	ch.SyncvarsByName[name] = ch.SyncvarsByID[id]
	ch.NameToID[name] = id
	resp := append(append([]byte{SYNCVAR_REGISTRY_CONFIRMATION, byte(len(name))}, []byte(name)...), cmp.Int16ToBytes(int16(id))...)
	ch.Server.Send(resp, ch.Server.ConnToIdx[ch.Conn])
	ch.idCounter++
	log.Printf("#####Server: Creating SyncVar with ID=%v, name='%s' , initiated by client=%s", id, name, ch.Conn.RemoteAddr().String())
}

func GetServerManager(s *Server) (sm *ServerManager) {
	sm = &ServerManager{Server: s, Handler: make(map[*ws.Conn]*ClientHandler), AllClients: make([]*ws.Conn, 0)}
	s.InputHandler = sm.receive
	s.OnNewConn = sm.onNewConn
	s.OnCloseConn = sm.onCloseConn
	return
}

type ServerManager struct {
	Server     *Server
	Handler    map[*ws.Conn]*ClientHandler
	AllClients []*ws.Conn

	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}

func (m *ServerManager) RegisterSyncVar(sv SyncVar, name string, clients ...*ws.Conn) {
	for _, c := range clients {
		m.Handler[c].RegisterSyncVar(sv, name)
	}
}
func (m *ServerManager) UpdateSyncVars() {
	for _, h := range m.Handler {
		h.UpdateSyncVars()
	}
}
func (m *ServerManager) DeleteSyncVar(name string, clients ...*ws.Conn) {
	for _, c := range clients {
		m.Handler[c].DeleteSyncVar(name)
	}
}
func (m *ServerManager) GetHandler(id int) *ClientHandler {
	if id >= 0 && id < len(m.AllClients) {
		return m.Handler[m.AllClients[id]]
	}
	return nil
}

func (m *ServerManager) receive(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	if err != nil {
		m.onCloseConn(c, mt, msg, err, s)
		return
	}
	if msg[0] == SYNCVAR_REGISTRY {
		m.Handler[c].processSyncVarRegistry(msg[1:])
	} else if msg[0] == SYNCVAR_UPDATE {
		m.Handler[c].onSyncVarUpdateC(msg[1:])
	} else if msg[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(msg[1:3]))
		name := string(msg[3:])
		m.Handler[c].deleteSyncVarLocal(name, id)
		log.Printf("#####Server: Deleting SyncVar with ID=%v, name='%s' , initiated by client=%s", id, name, c.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		go m.InputHandler(c, mt, msg, err, s)
	}
}
func (m *ServerManager) onNewConn(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	m.Handler[c] = GetNewClientHandler(m.Server, c)
	m.AllClients = append(m.AllClients, c)
	if m.OnNewConn != nil {
		go m.OnNewConn(c, mt, msg, err, s)
	}
}
func (m *ServerManager) onCloseConn(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	delete(m.Handler, c)
	m.AllClients = DeleteConn(true, c, m.AllClients...)
	if m.OnCloseConn != nil {
		go m.OnCloseConn(c, mt, msg, err, s)
	}
}
