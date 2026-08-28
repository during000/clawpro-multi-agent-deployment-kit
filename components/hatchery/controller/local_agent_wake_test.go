package controller

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func resetLocalAgentWakeStateForTest(t *testing.T) {
	t.Helper()
	localAgentWakeState.Lock()
	localAgentWakeState.tickets = make(map[string]localAgentWakeTicket)
	localAgentWakeState.connections = make(map[uint]map[*localAgentWakeConnection]struct{})
	localAgentWakeState.Unlock()
	t.Cleanup(func() {
		localAgentWakeState.Lock()
		for _, connections := range localAgentWakeState.connections {
			for connection := range connections {
				_ = connection.conn.Close()
			}
		}
		localAgentWakeState.tickets = make(map[string]localAgentWakeTicket)
		localAgentWakeState.connections = make(map[uint]map[*localAgentWakeConnection]struct{})
		localAgentWakeState.Unlock()
	})
}

func TestConsumeLocalAgentWakeTicketIsSingleUseAndExpires(t *testing.T) {
	resetLocalAgentWakeStateForTest(t)
	localAgentWakeState.Lock()
	localAgentWakeState.tickets["valid"] = localAgentWakeTicket{InstanceID: 7, ExpiresAt: time.Now().Add(time.Minute)}
	localAgentWakeState.tickets["expired"] = localAgentWakeTicket{InstanceID: 8, ExpiresAt: time.Now().Add(-time.Second)}
	localAgentWakeState.Unlock()

	ticket, ok := consumeLocalAgentWakeTicket("valid")
	if !ok || ticket.InstanceID != 7 {
		t.Fatalf("expected valid single-use ticket, got ok=%v ticket=%+v", ok, ticket)
	}
	if _, ok = consumeLocalAgentWakeTicket("valid"); ok {
		t.Fatal("ticket must not be reusable")
	}
	if _, ok = consumeLocalAgentWakeTicket("expired"); ok {
		t.Fatal("expired ticket must be rejected")
	}
}

func TestNotifyLocalAgentTaskAvailableWritesOnlyWakeMetadata(t *testing.T) {
	resetLocalAgentWakeStateForTest(t)
	server, client := net.Pipe()
	defer client.Close()
	connection := &localAgentWakeConnection{instanceID: 42, conn: server}
	registerLocalAgentWakeConnection(connection)

	type result struct {
		opcode  byte
		payload []byte
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		opcode, payload, err := readLocalAgentWakeFrame(bufio.NewReader(client))
		resultCh <- result{opcode: opcode, payload: payload, err: err}
	}()

	if delivered := notifyLocalAgentTaskAvailable(42, 99); delivered != 1 {
		t.Fatalf("expected one delivered wake, got %d", delivered)
	}
	frame := <-resultCh
	if frame.err != nil {
		t.Fatal(frame.err)
	}
	if frame.opcode != 0x1 {
		t.Fatalf("expected text frame, got opcode %d", frame.opcode)
	}
	var message map[string]any
	if err := json.Unmarshal(frame.payload, &message); err != nil {
		t.Fatal(err)
	}
	if message["type"] != "task_available" || message["task_id"] != float64(99) {
		t.Fatalf("unexpected wake payload: %s", frame.payload)
	}
	if _, hasPrompt := message["prompt"]; hasPrompt {
		t.Fatalf("wake payload must not contain prompt: %s", frame.payload)
	}
}
