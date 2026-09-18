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
	// Poll pages are meant to be shared publicly, so any origin can open the
	// results socket. Mutating actions (create/vote) still go through the
	// authenticated/validated REST endpoints, not this socket.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub keeps track of which WebSocket connections are watching which poll,
// and also subscribes to Redis pub/sub so results stay in sync even if
// multiple backend instances are running. Broadcasting only ever happens
// via the Redis-relayed path (listenRedis -> Broadcast), so every vote is
// delivered to each local client exactly once, and there is only ever one
// goroutine writing to a given connection.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool // pollID -> set of conns
	done    map[string]chan struct{}            // pollID -> stop signal for its listenRedis goroutine
	rdb     *redis.Client
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients: make(map[string]map[*websocket.Conn]bool),
		done:    make(map[string]chan struct{}),
		rdb:     rdb,
	}
}

func (h *Hub) Subscribe(pollID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[pollID] == nil {
		h.clients[pollID] = make(map[*websocket.Conn]bool)
		done := make(chan struct{})
		h.done[pollID] = done
		go h.listenRedis(pollID, done)
	}
	h.clients[pollID][conn] = true
}

func (h *Hub) Unsubscribe(pollID string, conn *websocket.Conn) {
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
// clients. This is what lets results stay live even across multiple backend
// replicas behind a load balancer. It stops cleanly as soon as the last local
// viewer disconnects (via done), rather than waiting on the next message.
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
// disconnects. It immediately sends the current snapshot so a viewer who
// joins mid-poll sees results right away, not just future updates.
func PollSocket(hub *Hub, pollHandler *PollHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}
		defer conn.Close()

		hub.Subscribe(pollID, conn)
		defer hub.Unsubscribe(pollID, conn)

		ctx := c.Request.Context()
		if results, err := pollHandler.currentResults(ctx, pollID); err == nil {
			if payload, err := json.Marshal(results); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, payload)
			}
		}

		// Keep reading (and discarding) to detect disconnects/pings.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}
