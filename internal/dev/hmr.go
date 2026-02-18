package dev

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type HMRMessage struct {
	Type    string `json:"type"`
	File    string `json:"file,omitempty"`
	Message string `json:"message,omitempty"`
}

type HMRSerer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]struct{}
	mu       sync.RWMutex
}

func NewHMRSerer() *HMRSerer {
	return &HMRSerer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (h *HMRSerer) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Debug("HMR WebSocket upgrade failed", "error", err)
			return
		}

		h.register(conn)
		defer h.unregister(conn)

		h.send(conn, HMRMessage{Type: "connected", Message: "HMR connected"})

		h.readPump(conn)
	}
}

func (h *HMRSerer) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
	slog.Debug("HMR client connected", "total", len(h.clients))
}

func (h *HMRSerer) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
	slog.Debug("HMR client disconnected", "total", len(h.clients))
}

func (h *HMRSerer) readPump(conn *websocket.Conn) {
	defer h.unregister(conn)

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Debug("HMR read error", "error", err)
			}
			break
		}
	}
}

func (h *HMRSerer) send(conn *websocket.Conn, msg HMRMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

func (h *HMRSerer) Broadcast(msg HMRMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal HMR message", "error", err)
		return
	}

	slog.Debug("Broadcasting HMR message", "type", msg.Type, "clients", len(h.clients))

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Debug("Failed to send HMR message", "error", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *HMRSerer) Reload(file string) {
	h.Broadcast(HMRMessage{
		Type:    "reload",
		File:    file,
		Message: "File changed, reloading...",
	})
}

func (h *HMRSerer) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		conn.Close()
	}
	h.clients = make(map[*websocket.Conn]struct{})
}
