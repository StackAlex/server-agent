package agent

import (
	"encoding/json"
	"log"
	"sync"

	"server-agent/internal/terminal"
	"server-agent/internal/websocket"
	"server-agent/protocol"
)

type TerminalManager struct {
	mu       sync.Mutex
	sessions map[string]*terminal.Terminal
}

func NewTerminalManager() *TerminalManager {
	return &TerminalManager{
		sessions: make(map[string]*terminal.Terminal),
	}
}

func (m *TerminalManager) Open(
	client *websocket.Client,
	requestID string,
	cols int,
	rows int,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, err := terminal.New(cols, rows)
	if err != nil {
		return err
	}

	m.sessions[t.ID] = t

	payload, err := json.Marshal(map[string]any{
		"session_id": t.ID,
	})
	if err != nil {
		delete(m.sessions, t.ID)
		_ = t.Close()
		return err
	}

	err = client.Send(protocol.Message{
		Type:      "terminal:opened",
		RequestID: requestID,
		Payload:   payload,
	})

	if err != nil {
		delete(m.sessions, t.ID)
		_ = t.Close()
		return err
	}

	log.Printf("Terminal opened: %s", t.ID)

	go m.readLoop(client, t)

	return nil
}

func (m *TerminalManager) readLoop(
	client *websocket.Client,
	t *terminal.Terminal,
) {
	buffer := make([]byte, 4096)

	for {
		n, err := t.Read(buffer)

		if n > 0 {
			payload, marshalErr := json.Marshal(map[string]string{
				"session_id": t.ID,
				"data":       string(buffer[:n]),
			})

			if marshalErr != nil {
				log.Printf(
					"terminal %s: failed to marshal output: %v",
					t.ID,
					marshalErr,
				)
				break
			}

			sendErr := client.Send(protocol.Message{
				Type:    "terminal:output",
				Payload: payload,
			})

			if sendErr != nil {
				log.Printf(
					"terminal %s: failed to send output: %v",
					t.ID,
					sendErr,
				)
				break
			}
		}

		if err != nil {
			log.Printf(
				"terminal %s read stopped: %v",
				t.ID,
				err,
			)
			break
		}
	}

	m.Close(client, t.ID, "terminal exited")
}

func (m *TerminalManager) Input(
	sessionID string,
	data string,
) error {
	m.mu.Lock()
	t, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		return nil
	}

	return t.Write([]byte(data))
}

func (m *TerminalManager) Resize(
	sessionID string,
	cols int,
	rows int,
) error {
	m.mu.Lock()
	t, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		return nil
	}

	return t.Resize(cols, rows)
}

func (m *TerminalManager) Close(
	client *websocket.Client,
	sessionID string,
	reason string,
) {
	m.mu.Lock()

	t, exists := m.sessions[sessionID]

	if exists {
		delete(m.sessions, sessionID)
	}

	m.mu.Unlock()

	if !exists {
		return
	}

	_ = t.Close()

	payload, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"reason":     reason,
	})

	_ = client.Send(protocol.Message{
		Type:    "terminal:close",
		Payload: payload,
	})

	log.Printf("Terminal closed: %s (%s)", sessionID, reason)
}
