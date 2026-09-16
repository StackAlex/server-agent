// internal/app/app.go

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"server-agent/internal/agent"
	"server-agent/internal/config"
	"server-agent/internal/monitor"
	"server-agent/internal/websocket"
	"server-agent/protocol"
)

type App struct {
	Config          *config.Config
	TerminalManager *agent.TerminalManager
}

func New() (*App, error) {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return nil, err
	}

	if cfg.Agent.UUID == "" {
		return nil, fmt.Errorf("agent UUID is required in config")
	}

	return &App{Config: cfg, TerminalManager: agent.NewTerminalManager()}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("Server Agent started")
	log.Println("Panel URL:", a.Config.Panel.URL)
	log.Println("Agent ID:", a.Config.Agent.UUID)

	// Инициализируем клиент ТОЛЬКО через New (без Dial)
	client := websocket.New(a.Config.Panel.URL)

	terminalManager := a.TerminalManager
	// Устанавливаем заголовки для WebSocket handshake

	log.Println("Connecting to panel...")
	if err := client.Connect(ctx); err != nil {
		log.Printf("Connect failed: %v", err)
		return fmt.Errorf("failed to connect to panel: %w", err)
	}
	log.Println("Connected to panel")

	payloadData := map[string]any{
		"agent_id":   a.Config.Agent.UUID,
		"agent_name": a.Config.Agent.Name,
		"token":      a.Config.Panel.Token,
	}

	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	// Отправляем auth-сообщение
	err = client.Send(protocol.Message{
		Type:    "auth",
		Payload: payloadBytes,
	})
	if err != nil {
		log.Printf("Failed to send auth message: %v", err)
		client.Close()
		return fmt.Errorf("auth send failed: %w", err)
	}
	log.Println("Auth message sent")

	// Дальше твои рабочие циклы
	go a.loop(ctx, client)
	go a.readLoop(ctx, client, terminalManager)

	<-ctx.Done()

	log.Println("Stopping Server Agent")
	client.Close()
	return nil
}

func (a *App) loop(ctx context.Context, client *websocket.Client) {
	ticker := time.NewTicker(time.Duration(a.Config.Monitor.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := monitor.Collect()
			if err != nil {
				log.Printf("collect metrics failed: %v", err)
				continue
			}

			payload, mErr := json.Marshal(map[string]any{
				"agent_id": a.Config.Agent.UUID,
				"stats":    stats,
			})
			if mErr != nil {
				log.Printf("marshal stats failed: %v", mErr)
				continue
			}

			msg := protocol.Message{
				Type:    "stats",
				Payload: payload,
			}

			if sErr := client.Send(msg); sErr != nil {
				log.Printf("send stats failed: %v", sErr)
				// соединение разорвано — можно выйти, переподключение лучше делать снаружи
				return
			}
		}
	}
}

func (a *App) readLoop(
	ctx context.Context,
	client *websocket.Client,
	terminalManager *agent.TerminalManager,
) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			data, err := client.Read()
			if err != nil {
				log.Printf("read failed: %v", err)
				return
			}

			var msg protocol.Message

			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("unmarshal message failed: %v", err)
				continue
			}

			log.Printf("received message: type=%s", msg.Type)

			switch msg.Type {

			case "auth_ok":
				var payload struct {
					AgentID string `json:"agent_id"`
				}

				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					log.Printf("auth_ok payload error: %v", err)
					continue
				}

				log.Printf(
					"Authentication successful! Agent ID: %s",
					payload.AgentID,
				)

			case "command.run":
				log.Printf("Command received: %s", string(msg.Payload))

			case "terminal:open":
				var payload protocol.TerminalOpenPayload

				if err := protocol.Decode(msg.Payload, &payload); err != nil {
					log.Printf("terminal:open payload error: %v", err)
					continue
				}

				if payload.Cols <= 0 {
					payload.Cols = 120
				}

				if payload.Rows <= 0 {
					payload.Rows = 30
				}

				if err := terminalManager.Open(
					client,
					msg.RequestID,
					payload.Cols,
					payload.Rows,
				); err != nil {
					log.Printf("terminal open failed: %v", err)
				}

			case "terminal:input":
				var payload struct {
					SessionID string `json:"session_id"`
					Data      string `json:"data"`
				}

				if err := protocol.Decode(msg.Payload, &payload); err != nil {
					log.Printf("terminal:input payload error: %v", err)
					continue
				}

				if err := terminalManager.Input(
					payload.SessionID,
					payload.Data,
				); err != nil {
					log.Printf("terminal input failed: %v", err)
				}

			case "terminal:resize":
				var payload struct {
					SessionID string `json:"session_id"`
					Cols      int    `json:"cols"`
					Rows      int    `json:"rows"`
				}

				if err := protocol.Decode(msg.Payload, &payload); err != nil {
					log.Printf("terminal:resize payload error: %v", err)
					continue
				}

				if err := terminalManager.Resize(
					payload.SessionID,
					payload.Cols,
					payload.Rows,
				); err != nil {
					log.Printf("terminal resize failed: %v", err)
				}

			case "terminal:close":
				var payload struct {
					SessionID string `json:"session_id"`
					Reason    string `json:"reason"`
				}

				if err := protocol.Decode(msg.Payload, &payload); err != nil {
					log.Printf("terminal:close payload error: %v", err)
					continue
				}

				terminalManager.Close(
					client,
					payload.SessionID,
					payload.Reason,
				)

			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
		}
	}
}
