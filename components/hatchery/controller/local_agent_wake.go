package controller

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"gorm.io/gorm"
)

const (
	localAgentWakeTicketTTL = 60 * time.Second
	maxWakeFrameBytes       = 16 << 10
)

type localAgentWakeTicket struct {
	UserID      uint
	InstanceID  uint
	InstanceCID string
	ExpiresAt   time.Time
}

type localAgentWakeConnection struct {
	instanceID uint
	conn       net.Conn
	writeMu    sync.Mutex
}

var localAgentWakeState = struct {
	sync.Mutex
	tickets     map[string]localAgentWakeTicket
	connections map[uint]map[*localAgentWakeConnection]struct{}
}{
	tickets:     make(map[string]localAgentWakeTicket),
	connections: make(map[uint]map[*localAgentWakeConnection]struct{}),
}

type localAgentWakeTicketRequest struct {
	AgentType    string `json:"agent_type"`
	LocalAgentID string `json:"local_agent_id"`
}

// HandleLocalAgentWakeTicket uses the authenticated HTTPS channel to mint a
// short-lived, single-use ticket. The ticket keeps bearer credentials out of
// the WebSocket URL and mirrors the production outbound-connection design.
func HandleLocalAgentWakeTicket(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) {
		return
	}

	var req localAgentWakeTicketRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nRichError(err, i18n.MsgBadRequest))
		return
	}
	instanceCID, validationErr := validateLocalAgentInputs(
		strings.TrimSpace(req.AgentType),
		strings.TrimSpace(req.LocalAgentID),
	)
	if validationErr != nil {
		writeError(w, r, http.StatusBadRequest, validationErr)
		return
	}

	var inst model.Instance
	if err := model.DB(r.Context()).Where(
		"user_id = ? AND instance_id = ? AND source = ?",
		user.ID, instanceCID, model.InstanceSourceLocal,
	).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, r, http.StatusNotFound, hcommon.I18nError(i18n.MsgInstanceNotFoundOrNoPerm))
			return
		}
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}

	ticketBytes := make([]byte, 24)
	if _, err := rand.Read(ticketBytes); err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgInternalError))
		return
	}
	ticket := base64.RawURLEncoding.EncodeToString(ticketBytes)
	expiresAt := time.Now().Add(localAgentWakeTicketTTL)
	localAgentWakeState.Lock()
	for key, item := range localAgentWakeState.tickets {
		if time.Now().After(item.ExpiresAt) {
			delete(localAgentWakeState.tickets, key)
		}
	}
	localAgentWakeState.tickets[ticket] = localAgentWakeTicket{
		UserID: user.ID, InstanceID: inst.ID, InstanceCID: inst.InstanceId, ExpiresAt: expiresAt,
	}
	localAgentWakeState.Unlock()

	jsonOK(w, map[string]any{
		"ok": true, "ticket": ticket, "expires_in_seconds": int(localAgentWakeTicketTTL.Seconds()),
	})
}

func consumeLocalAgentWakeTicket(raw string) (localAgentWakeTicket, bool) {
	localAgentWakeState.Lock()
	defer localAgentWakeState.Unlock()
	ticket, ok := localAgentWakeState.tickets[raw]
	if ok {
		delete(localAgentWakeState.tickets, raw)
	}
	if !ok || time.Now().After(ticket.ExpiresAt) {
		return localAgentWakeTicket{}, false
	}
	return ticket, true
}

// HandleLocalAgentWake upgrades an outbound TeamAI connection to WebSocket.
// Full task contents never travel here: messages only announce that a task is
// available, and TeamAI claims it through the authenticated sync endpoint.
func HandleLocalAgentWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
		!strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return
	}
	ticket, ok := consumeLocalAgentWakeTicket(strings.TrimSpace(r.URL.Query().Get("ticket")))
	if !ok {
		http.Error(w, "invalid or expired wake ticket", http.StatusUnauthorized)
		return
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket upgrade unavailable", http.StatusInternalServerError)
		return
	}
	conn, buffered, err := hijacker.Hijack()
	if err != nil {
		return
	}
	accept := computeWebSocketAccept(key)
	if _, err = fmt.Fprintf(
		buffered,
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		accept,
	); err != nil {
		_ = conn.Close()
		return
	}
	if err = buffered.Flush(); err != nil {
		_ = conn.Close()
		return
	}

	wakeConn := &localAgentWakeConnection{instanceID: ticket.InstanceID, conn: conn}
	registerLocalAgentWakeConnection(wakeConn)
	defer func() {
		unregisterLocalAgentWakeConnection(wakeConn)
		_ = conn.Close()
	}()
	_ = wakeConn.sendJSON(map[string]any{
		"type": "sync_required", "instance_id": ticket.InstanceCID, "reason": "connected",
	})
	wakeConn.readLoop(buffered.Reader)
}

func registerLocalAgentWakeConnection(conn *localAgentWakeConnection) {
	localAgentWakeState.Lock()
	defer localAgentWakeState.Unlock()
	connections := localAgentWakeState.connections[conn.instanceID]
	if connections == nil {
		connections = make(map[*localAgentWakeConnection]struct{})
		localAgentWakeState.connections[conn.instanceID] = connections
	}
	connections[conn] = struct{}{}
}

func unregisterLocalAgentWakeConnection(conn *localAgentWakeConnection) {
	localAgentWakeState.Lock()
	defer localAgentWakeState.Unlock()
	connections := localAgentWakeState.connections[conn.instanceID]
	delete(connections, conn)
	if len(connections) == 0 {
		delete(localAgentWakeState.connections, conn.instanceID)
	}
}

// notifyLocalAgentTaskAvailable returns the number of live TeamAI connections
// that accepted the lightweight wake event. A zero count is safe: the task is
// already durable and will be returned by the next sync after reconnection.
func notifyLocalAgentTaskAvailable(instanceID, taskID uint) int {
	localAgentWakeState.Lock()
	connections := make([]*localAgentWakeConnection, 0, len(localAgentWakeState.connections[instanceID]))
	for conn := range localAgentWakeState.connections[instanceID] {
		connections = append(connections, conn)
	}
	localAgentWakeState.Unlock()

	delivered := 0
	for _, conn := range connections {
		if err := conn.sendJSON(map[string]any{
			"type": "task_available", "task_id": taskID, "issued_at": time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			unregisterLocalAgentWakeConnection(conn)
			_ = conn.conn.Close()
			continue
		}
		delivered++
	}
	return delivered
}

func (c *localAgentWakeConnection) sendJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.writeFrame(0x1, payload)
}

func (c *localAgentWakeConnection) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > maxWakeFrameBytes {
		return fmt.Errorf("wake frame exceeds %d bytes", maxWakeFrameBytes)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	header := []byte{0x80 | opcode}
	switch {
	case len(payload) <= 125:
		header = append(header, byte(len(payload)))
	case len(payload) <= 65535:
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	default:
		return fmt.Errorf("wake frame too large")
	}
	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(payload)
	return err
}

func (c *localAgentWakeConnection) readLoop(reader *bufio.Reader) {
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(75 * time.Second))
		opcode, payload, err := readLocalAgentWakeFrame(reader)
		if err != nil {
			return
		}
		switch opcode {
		case 0x1:
			var message struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &message) == nil && message.Type == "heartbeat" {
				_ = c.sendJSON(map[string]any{"type": "heartbeat_ack", "at": time.Now().UTC().Format(time.RFC3339Nano)})
			}
		case 0x8:
			_ = c.writeFrame(0x8, nil)
			return
		case 0x9:
			_ = c.writeFrame(0xA, payload)
		}
	}
}

func readLocalAgentWakeFrame(reader io.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7F)
	if length == 126 {
		buf := make([]byte, 2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	} else if length == 127 {
		buf := make([]byte, 8)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(buf)
	}
	if length > maxWakeFrameBytes {
		return 0, nil, fmt.Errorf("wake frame exceeds %d bytes", maxWakeFrameBytes)
	}
	mask := make([]byte, 4)
	if masked {
		if _, err := io.ReadFull(reader, mask); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for index := range payload {
			payload[index] ^= mask[index%4]
		}
	}
	return opcode, payload, nil
}
