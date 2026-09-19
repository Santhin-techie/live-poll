package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// safeConn wraps a websocket connection with a mutex so writes from
// different goroutines (a fresh-snapshot send on connect vs. a broadcast
// triggered by another user's vote) can never race against each other.
// gorilla/websocket panics on concurrent writes to the same connection,
// which is exactly the bug this fixes.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeConn) WriteMessage(messageType int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(messageType, data)
}

// Hub keeps track of which WebSocket connections are watching which poll,
// and also subscribes to Redis pub/sub so results stay in sync even if
// multiple backend instances are running.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*safeConn]bool // pollID -> set of conns
	done    map[string]chan struct{}      // pollID -> stop signal for its listenRedis goroutine
	rdb     *redis.Client
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients: make(map[string]map[*safeConn]bool),
		done:    make(map[string]chan struct{}),
		rdb:     rdb,
	}
}

func (h *Hub) Subscribe(pollID string, conn *safeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[pollID] == nil {
		h.clients[pollID] = make(map[*safeConn]bool)
		done := make(chan struct{})
		h.done[pollID] = done
		go h.listenRedis(pollID, done)
	}
	h.clients[pollID][conn] = true
}

func (h *Hub) Unsubscribe(pollID string, conn *safeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[pollID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, pollID)
			if done, ok := h.done[pollID]; ok {
				close(done)
				delete(h.done, pollID)
			}
		}
	}
}

// Broadcast pushes a payload to every client on this instance watching pollID.
func (h *Hub) Broadcast(pollID string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients[pollID] {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("ws write error: %v", err)
		}
	}
}

// listenRedis relays messages published on this poll's Redis channel to local
// clients, stopping cleanly as soon as the last local viewer disconnects.
func (h *Hub) listenRedis(pollID string, done chan struct{}) {
	ctx := context.Background()
	sub := h.rdb.Subscribe(ctx, redisChannel(pollID))
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-done:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.Broadcast(pollID, []byte(msg.Payload))
		}
	}
}

// PollSocket upgrades the connection and keeps it registered until the client
// disconnects. It sends the current snapshot before subscribing, so there is
// no window where a concurrent broadcast could race the initial write.
func PollSocket(hub *Hub, pollHandler *PollHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")

		rawConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}
		defer rawConn.Close()

		conn := &safeConn{conn: rawConn}

		// Send the initial snapshot BEFORE subscribing, so the Hub can't try
		// to write to this connection until this first write is done.
		ctx := c.Request.Context()
		if results, err := pollHandler.currentResults(ctx, pollID); err == nil {
			if payload, err := json.Marshal(results); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, payload)
			}
		}

		hub.Subscribe(pollID, conn)
		defer hub.Unsubscribe(pollID, conn)

		// Keep reading (and discarding) to detect disconnects/pings.
		for {
			if _, _, err := rawConn.ReadMessage(); err != nil {
				break
			}
		}
	}
}