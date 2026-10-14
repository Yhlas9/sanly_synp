// Package hub рассылает события занятия по локальной сети: состояние урока,
// подключения учеников и кадры демонстрации экрана учителя.
package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	RoleTeacher = "teacher"
	RoleStudent = "student"

	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	// Кадр демонстрации экрана 720p в JPEG укладывается с запасом.
	maxMessage = 2 << 20
)

// Event — сообщение, уходящее на устройства.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	SessionID int64
	Role      string
	MemberID  int64 // participants.id для ученика, teachers.id для учителя
}

type Hub struct {
	mu       sync.RWMutex
	sessions map[int64]map[*Client]struct{}
	// OnStudentPresence вызывается при подключении/отключении ученика.
	OnStudentPresence func(participantID int64, online bool)
	// OnTeacherFrame — кадр демонстрации экрана от учителя.
	OnTeacherFrame func(sessionID int64, payload []byte)
}

func New() *Hub {
	return &Hub{sessions: make(map[int64]map[*Client]struct{})}
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	if h.sessions[c.SessionID] == nil {
		h.sessions[c.SessionID] = make(map[*Client]struct{})
	}
	h.sessions[c.SessionID][c] = struct{}{}
	h.mu.Unlock()

	if c.Role == RoleStudent && h.OnStudentPresence != nil {
		h.OnStudentPresence(c.MemberID, true)
	}
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	if set, ok := h.sessions[c.SessionID]; ok {
		if _, found := set[c]; found {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.sessions, c.SessionID)
		}
	}
	h.mu.Unlock()

	if c.Role == RoleStudent && h.OnStudentPresence != nil {
		h.OnStudentPresence(c.MemberID, false)
	}
}

// Broadcast рассылает событие всем в занятии; roles пуст — значит всем.
func (h *Hub) Broadcast(sessionID int64, ev Event, roles ...string) {
	payload, err := json.Marshal(ev)
	if err != nil {
		log.Printf("hub: marshal %s: %v", ev.Type, err)
		return
	}
	h.BroadcastRaw(sessionID, payload, roles...)
}

func (h *Hub) BroadcastRaw(sessionID int64, payload []byte, roles ...string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.sessions[sessionID] {
		if len(roles) > 0 && !contains(roles, c.Role) {
			continue
		}
		select {
		case c.send <- payload:
		default:
			// Медленный клиент не тормозит остальных: кадр для него теряется.
		}
	}
}

// CountStudents — сколько учеников сейчас на связи.
func (h *Hub) CountStudents(sessionID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	n := 0
	for c := range h.sessions[sessionID] {
		if c.Role == RoleStudent {
			n++
		}
	}
	return n
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// Serve берёт уже поднятое соединение и обслуживает его до разрыва.
func (h *Hub) Serve(conn *websocket.Conn, sessionID int64, role string, memberID int64) {
	c := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 16),
		SessionID: sessionID,
		Role:      role,
		MemberID:  memberID,
	}
	h.add(c)

	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.remove(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessage)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		// Кадры демонстрации экрана шлёт только учитель.
		if c.Role == RoleTeacher && c.hub.OnTeacherFrame != nil {
			c.hub.OnTeacherFrame(c.SessionID, msg)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
