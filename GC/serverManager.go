package GC

import (
	"sync"

	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/Compression"
)

type Handler interface {
	RegisterSyncVarNormal(sv SyncVar, ACID int)
	RegisterSyncVarBuffered(sv SyncVar, ACID int)
	RegisterSyncVarsNormal(svs map[int]SyncVar)
	RegisterSyncVarsBuffered(svs map[int]SyncVar)
	UpdateSyncVarsWithACIDsNormal(...int) int
	UpdateSyncVarsWithACIDsBuffered(...int) int
	UpdateSyncVarsNormal() int
	UpdateSyncVarsBuffered() int
	DeleteSyncVarNormal(ACID int)
	DeleteSyncVarBuffered(ACID int)
	RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int))
}

func GetNewClientHandler(s *Server, cn *ws.Conn) (ch *ClientHandler) {
	ch = &ClientHandler{Server: s, Conn: cn}
	ch.idCounter = 0
	ch.SyncvarsByID = make(map[int]SyncVar)
	ch.SyncvarsByACID = make(map[int]SyncVar)
	ch.ACIDToID = make(map[int]int)
	ch.SyncVarOnChange = make(map[int]func(SyncVar, int))
	return
}

type ClientHandler struct {
	Server                                 *Server
	Conn                                   *ws.Conn
	SyncvarsByID                           map[int]SyncVar
	SyncvarsByACID                         map[int]SyncVar
	ACIDToID                               map[int]int
	SyncVarOnChange                        map[int]func(SyncVar, int)
	SV_ID_L, SV_ACID_L, ACID_ID_L, SV_chng sync.Mutex
	idCounter                              int
}

//you may wait for confirmation
func (ch *ClientHandler) RegisterSyncVarNormal(sv SyncVar, ACID int) {
	ch.Server.SendNormal(ch.registerSyncVar(sv, ACID), ch.Conn)
	ch.idCounter++
}
func (ch *ClientHandler) RegisterSyncVarBuffered(sv SyncVar, ACID int) {
	ch.Server.SendBuffered(ch.registerSyncVar(sv, ACID), ch.Conn)
	ch.idCounter++
}
func (ch *ClientHandler) registerSyncVar(sv SyncVar, ACID int) []byte {
	printLogF(3, "#####Server: Creating SyncVar with ID=%v, ACID='%v', type=%v , on conn %p\n", ch.idCounter, ACID, sv.Type(), ch.Conn)
	ch.SV_ACID_L.Lock()
	ch.SyncvarsByACID[ACID] = sv
	ch.SV_ACID_L.Unlock()
	ch.SV_ID_L.Lock()
	ch.SyncvarsByID[ch.idCounter] = sv
	ch.SV_ID_L.Unlock()
	ch.ACID_ID_L.Lock()
	ch.ACIDToID[ACID] = ch.idCounter
	ch.ACID_ID_L.Unlock()
	return append(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, cmp.Int16ToBytes(int16(ACID))...), cmp.Int16ToBytes(int16(ch.idCounter))...)
}

//you may wait for confirmation
func (ch *ClientHandler) RegisterSyncVarsNormal(svs map[int]SyncVar) {
	ch.Server.SendNormal(ch.registerSyncVars(svs), ch.Conn)
}
func (ch *ClientHandler) RegisterSyncVarsBuffered(svs map[int]SyncVar) {
	ch.Server.SendBuffered(ch.registerSyncVars(svs), ch.Conn)
}
func (ch *ClientHandler) registerSyncVars(svs map[int]SyncVar) []byte {
	data := []byte{SYNCVAR_M_REGISTRY}
	for ACID, sv := range svs {
		//printLogF("#####Server: MCreating SyncVar with ID=%v, ACID='%v', type=%v , initiated by self=%s", ch.idCounter, ACID, sv.Type(), ch.Conn.LocalAddr().String())
		ch.SV_ACID_L.Lock()
		ch.SyncvarsByACID[ACID] = sv
		ch.SV_ACID_L.Unlock()
		ch.SV_ID_L.Lock()
		ch.SyncvarsByID[ch.idCounter] = sv
		ch.SV_ID_L.Unlock()
		ch.ACID_ID_L.Lock()
		ch.ACIDToID[ACID] = ch.idCounter
		ch.ACID_ID_L.Unlock()
		data = append(data, sv.Type())
		data = append(data, cmp.Int16ToBytes(int16(ACID))...)
		data = append(data, cmp.Int16ToBytes(int16(ch.idCounter))...)
		ch.idCounter++
	}
	printLogF(3, "#####Server: MCreating %v SyncVars, on conn %p\n", len(svs), ch.Conn)
	return data
}
func (ch *ClientHandler) RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int)) {
	ch.ACID_ID_L.Lock()
	id, ok := ch.ACIDToID[ACID]
	ch.ACID_ID_L.Unlock()
	if ok {
		ch.SV_chng.Lock()
		ch.SyncVarOnChange[id] = fnc
		ch.SV_chng.Unlock()
	}
}

//you may wait for confirmation
func (ch *ClientHandler) UpdateSyncVarsWithACIDsNormal(ACIDs ...int) (uc int) {
	var_data, count := ch.updateSyncVarsByACIDAndReturnData(false, ACIDs...)
	uc = count
	if uc > 0 {
		printLogF(2, "#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.SendNormal(var_data, ch.Conn)
	}
	return
}
func (ch *ClientHandler) UpdateSyncVarsWithACIDsBuffered(ACIDs ...int) (uc int) {
	var_data, count := ch.updateSyncVarsByACIDAndReturnData(false, ACIDs...)
	uc = count
	if uc > 0 {
		printLogF(2, "#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.SendBuffered(var_data, ch.Conn)
	}
	return
}

//you may wait for confirmation
func (ch *ClientHandler) UpdateSyncVarsNormal() (uc int) {
	var_data, count := ch.updateSyncVarsByIDAndReturnData(true)
	uc = count
	if uc > 0 {
		printLogF(2, "#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.SendNormal(var_data, ch.Conn)
	}
	return
}
func (ch *ClientHandler) UpdateSyncVarsBuffered() (uc int) {
	var_data, count := ch.updateSyncVarsByIDAndReturnData(true)
	uc = count
	if uc > 0 {
		printLogF(2, "#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.SendBuffered(var_data, ch.Conn)
	}
	return
}
func (ch *ClientHandler) updateSyncVarsByACIDAndReturnData(all bool, ACIDs ...int) (var_data []byte, uc int) {
	var_data = []byte{SYNCVAR_UPDATE}
	ch.SV_ACID_L.Lock()
	for ACID, sv := range ch.SyncvarsByACID {
		if sv.IsDirty() && (all || containsI(ACIDs, ACID)) {
			uc++
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			ch.ACID_ID_L.Lock()
			id := int16(ch.ACIDToID[ACID])
			ch.ACID_ID_L.Unlock()
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
			//printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), ch.Conn.LocalAddr().String())
		}
	}
	ch.SV_ACID_L.Unlock()
	return
}
func (ch *ClientHandler) updateSyncVarsByIDAndReturnData(all bool, IDs ...int) (var_data []byte, uc int) {
	var_data = []byte{SYNCVAR_UPDATE}
	ch.SV_ID_L.Lock()
	for id, sv := range ch.SyncvarsByID {
		if sv.IsDirty() && (all || containsI(IDs, id)) {
			uc++
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
			//printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), ch.Conn.LocalAddr().String())
		}
	}
	ch.SV_ID_L.Unlock()
	return
}

//you may wait for confirmation
func (ch *ClientHandler) DeleteSyncVarNormal(ACID int) {
	ch.ACID_ID_L.Lock()
	id := ch.ACIDToID[ACID]
	ch.ACID_ID_L.Unlock()
	printLogF(3, "#####Server: Deleting SyncVar with ID=%v, ACID='%v', on conn %p\n", id, ACID, ch.Conn)
	ch.deleteSyncVarLocal(ACID, id)
	ch.deleteSyncVarRemoteNormal(ACID, id)
}
func (ch *ClientHandler) DeleteSyncVarBuffered(ACID int) {
	ch.ACID_ID_L.Lock()
	id := ch.ACIDToID[ACID]
	ch.ACID_ID_L.Unlock()
	printLogF(3, "#####Server: Deleting SyncVar with ID=%v, ACID='%v', on conn %p\n", id, ACID, ch.Conn)
	ch.deleteSyncVarLocal(ACID, id)
	ch.deleteSyncVarRemoteBuffered(ACID, id)
}
func (ch *ClientHandler) deleteSyncVarLocal(ACID int, id int) {
	ch.SV_ID_L.Lock()
	delete(ch.SyncvarsByID, id)
	ch.SV_ID_L.Unlock()
	ch.SV_ACID_L.Lock()
	delete(ch.SyncvarsByACID, ACID)
	ch.SV_ACID_L.Unlock()
	ch.ACID_ID_L.Lock()
	delete(ch.ACIDToID, ACID)
	ch.ACID_ID_L.Unlock()
}
func (ch *ClientHandler) deleteSyncVarRemoteBuffered(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	ch.Server.SendBuffered(append([]byte{SYNCVAR_DELETION}, data...), ch.Conn)
}
func (ch *ClientHandler) deleteSyncVarRemoteNormal(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	ch.Server.SendNormal(append([]byte{SYNCVAR_DELETION}, data...), ch.Conn)
}
func (ch *ClientHandler) onSyncVarUpdateC(data []byte) {
	num := 0
	for true {
		num++
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4 : 4+l]
		data = data[4+l:]
		ch.SV_ID_L.Lock()
		sv := ch.SyncvarsByID[id]
		ch.SV_ID_L.Unlock()
		sv.SetData(dat)
		ch.SV_chng.Lock()
		if fnc, ok := ch.SyncVarOnChange[id]; ok {
			defer fnc(sv, id)
		}
		ch.SV_chng.Unlock()
		if len(data) <= 0 {
			break
		}
		//printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by client=%s", id, l, ch.Conn.RemoteAddr().String())
	}
	printLogF(2, "#####Server: Updating %v SyncVars, from conn %p\n", num, ch.Conn)
}
func (ch *ClientHandler) processSyncVarRegistry(data []byte) {
	t := data[0]
	ACID := int(cmp.BytesToInt16(data[1:3]))
	id := ch.idCounter
	ch.SV_ID_L.Lock()
	ch.SyncvarsByID[id] = GetSyncVarOfType(t)
	ch.SV_ACID_L.Lock()
	ch.SyncvarsByACID[ACID] = ch.SyncvarsByID[id]
	ch.SV_ACID_L.Unlock()
	ch.SV_ID_L.Unlock()
	ch.ACID_ID_L.Lock()
	ch.ACIDToID[ACID] = id
	ch.ACID_ID_L.Unlock()
	resp := append(append([]byte{SYNCVAR_REGISTRY_CONFIRMATION}, data[1:3]...), cmp.Int16ToBytes(int16(id))...)
	ch.Server.SendBuffered(resp, ch.Conn)
	ch.idCounter++
	printLogF(3, "#####Server: Creating SyncVar with ID=%v, ACID='%v', from conn %p\n", id, ACID, ch.Conn)
}

func GetServerManager(s *Server) (sm *ServerManager) {
	sm = &ServerManager{Server: s, Handler: make(map[*ws.Conn]*ClientHandler), AllClients: make([]*ws.Conn, 0)}
	s.InputHandler = sm.receive
	s.OnNewConn = sm.onNewConn
	s.OnCloseConn = sm.onCloseConn
	sm.StandardOnChange = nil
	return
}

type ServerManager struct {
	Server           *Server
	Handler          map[*ws.Conn]*ClientHandler
	AllClients       []*ws.Conn
	StandardOnChange func(SyncVar, int)

	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}

func (m *ServerManager) RegisterSyncVarToAllClients(normal bool, sv SyncVar, ACID int) {
	m.RegisterSyncVar(normal, sv, ACID, m.AllClients...)
}
func (m *ServerManager) RegisterSyncVarsToAllClients(normal bool, svs map[int]SyncVar) {
	m.RegisterSyncVars(normal, svs, m.AllClients...)
}

//you may wait for confirmation if normal
func (m *ServerManager) RegisterSyncVars(normal bool, svs map[int]SyncVar, clients ...*ws.Conn) {
	for _, sv := range svs {
		sv.IsRegisteredTo(len(clients))
	}
	for _, c := range clients {
		if normal {
			m.Handler[c].RegisterSyncVarsNormal(svs)
		} else {
			m.Handler[c].RegisterSyncVarsBuffered(svs)
		}
		if m.StandardOnChange != nil {
			for ACID := range svs {
				m.Handler[c].RegisterOnChangeFunc(ACID, m.StandardOnChange)
			}
		}
	}
}

//you may wait for confirmation if normal
func (m *ServerManager) RegisterSyncVar(normal bool, sv SyncVar, ACID int, clients ...*ws.Conn) {
	sv.IsRegisteredTo(len(clients))
	for _, c := range clients {
		if normal {
			m.Handler[c].RegisterSyncVarNormal(sv, ACID)
		} else {
			m.Handler[c].RegisterSyncVarBuffered(sv, ACID)
		}
		if m.StandardOnChange != nil {
			m.Handler[c].RegisterOnChangeFunc(ACID, m.StandardOnChange)
		}
	}
}
func (m *ServerManager) RegisterOnChangeFunc(ACID int, fncs []func(SyncVar, int), clients ...*ws.Conn) {
	fnc := fncs[0]
	for i, c := range clients {
		if i > 0 && i < len(fncs) {
			fnc = fncs[i]
		}
		m.Handler[c].RegisterOnChangeFunc(ACID, fnc)
	}
}
func (m *ServerManager) RegisterOnChangeFuncToAllClients(ACID int, fnc func(SyncVar, int)) {
	for _, c := range m.AllClients {
		m.Handler[c].RegisterOnChangeFunc(ACID, fnc)
	}
}

//you may wait for confirmation
func (m *ServerManager) UpdateSyncVarsNormal() {
	for _, h := range m.Handler {
		h.UpdateSyncVarsNormal()
	}
}
func (m *ServerManager) UpdateSyncVarsBuffered() {
	for _, h := range m.Handler {
		h.UpdateSyncVarsBuffered()
	}
}

//you may wait for confirmation
func (m *ServerManager) DeleteSyncVarNormal(ACID int, clients ...*ws.Conn) {
	for _, c := range clients {
		m.Handler[c].DeleteSyncVarNormal(ACID)
	}
}
func (m *ServerManager) DeleteSyncVarBuffered(ACID int, clients ...*ws.Conn) {
	for _, c := range clients {
		m.Handler[c].DeleteSyncVarBuffered(ACID)
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
		ACID := int(cmp.BytesToInt16(msg[3:5]))
		m.Handler[c].deleteSyncVarLocal(ACID, id)
		printLogF(3, "#####Server: Deleting SyncVar with ID=%v, ACID='%v', from conn %p\n", id, ACID, c)
	}
	if m.InputHandler != nil {
		m.InputHandler(c, mt, msg, err, s)
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
