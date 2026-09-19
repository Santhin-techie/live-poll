package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"pollapp/models"
)

type PollHandler struct {
	Polls *mongo.Collection
	Votes *mongo.Collection
	Redis *redis.Client
	Hub   *Hub
}

func NewPollHandler(polls, votes *mongo.Collection, rdb *redis.Client, hub *Hub) *PollHandler {
	return &PollHandler{Polls: polls, Votes: votes, Redis: rdb, Hub: hub}
}

type createPollRequest struct {
	Question   string   `json:"question" binding:"required,min=3,max=300"`
	Options    []string `json:"options" binding:"required,min=2,max=10,dive,required,min=1,max=120"`
	AllowMulti bool     `json:"allow_multi"`
}

func redisCountsKey(pollID string) string {
	return fmt.Sprintf("poll:%s:counts", pollID)
}

func redisChannel(pollID string) string {
	return fmt.Sprintf("poll:%s:updates", pollID)
}

// CreatePoll validates the payload server-side (never trusts the client),
// persists the poll in Mongo, and seeds its live counters in Redis at 0.
func (h *PollHandler) CreatePoll(c *gin.Context) {
	var req createPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDHex := c.GetString("userID")
	ownerID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// De-dupe / trim options server-side; never trust client-shaped IDs.
	options := make([]models.Option, 0, len(req.Options))
	seen := map[string]bool{}
	for i, text := range req.Options {
		text = strings.TrimSpace(text)
		if text == "" || seen[strings.ToLower(text)] {
			continue
		}
		seen[strings.ToLower(text)] = true
		options = append(options, models.Option{
			ID:   fmt.Sprintf("opt_%d", i+1),
			Text: text,
		})
	}
	if len(options) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a poll needs at least 2 distinct options"})
		return
	}

	poll := models.Poll{
		Question:   strings.TrimSpace(req.Question),
		Options:    options,
		OwnerID:    ownerID,
		AllowMulti: req.AllowMulti,
		CreatedAt:  time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	res, err := h.Polls.InsertOne(ctx, poll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create poll"})
		return
	}
	poll.ID = res.InsertedID.(primitive.ObjectID)

	// Seed Redis hash so GET /polls/:id can always read counts from Redis,
	// even before the first vote comes in.
	countsKey := redisCountsKey(poll.ID.Hex())
	fields := make(map[string]interface{})
	for _, o := range options {
		fields[o.ID] = 0
	}
	if err := h.Redis.HSet(ctx, countsKey, fields).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not initialize live counters"})
		return
	}

	c.JSON(http.StatusCreated, poll)
}

// GetPoll returns poll metadata plus its live counts, read from Redis.
func (h *PollHandler) GetPoll(c *gin.Context) {
	pollID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	if err := h.Polls.FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}

	results, err := h.currentResults(ctx, pollID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load live results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"poll": poll, "results": results})
}

// ListMyPolls returns polls owned by the authenticated user (their dashboard),
// including each poll's current total vote count read from Redis.
func (h *PollHandler) ListMyPolls(c *gin.Context) {
	userIDHex := c.GetString("userID")
	ownerID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cursor, err := h.Polls.Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load polls"})
		return
	}
	defer cursor.Close(ctx)

	polls := []models.Poll{}
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load polls"})
		return
	}

	type pollWithStats struct {
		models.Poll `bson:",inline"`
		TotalVotes  int `json:"total_votes"`
	}

	out := make([]pollWithStats, 0, len(polls))
	for _, p := range polls {
		total := 0
		if results, err := h.currentResults(ctx, p.ID.Hex()); err == nil {
			total = results.Total
		}
		out = append(out, pollWithStats{Poll: p, TotalVotes: total})
	}

	c.JSON(http.StatusOK, out)
}

// ClosePoll marks a poll closed so it stops accepting votes. Only the poll's
// owner can do this — verified against the authenticated user, not trusted
// from the request body.
func (h *PollHandler) ClosePoll(c *gin.Context) {
	pollID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	userIDHex := c.GetString("userID")
	ownerID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	res, err := h.Polls.UpdateOne(ctx,
		bson.M{"_id": objID, "owner_id": ownerID},
		bson.M{"$set": bson.M{"is_closed": true}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close poll"})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found or not yours"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": true})
}

type voteRequest struct {
	OptionID string `json:"option_id" binding:"required"`
}

// Vote is the hot path: it validates the option against the poll stored in
// Mongo (never trusts the client-supplied option id blindly), records the
// vote for history/de-duplication, then does the *live* part entirely in
// Redis: HINCRBY the counter and PUBLISH the new snapshot to subscribers.
func (h *PollHandler) Vote(c *gin.Context) {
	pollID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	var req voteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	if err := h.Polls.FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}
	if poll.IsClosed {
		c.JSON(http.StatusConflict, gin.H{"error": "this poll is closed"})
		return
	}

	// Validate the option actually belongs to this poll — never trust the client.
	valid := false
	for _, o := range poll.Options {
		if o.ID == req.OptionID {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid option for this poll"})
		return
	}

	// Basic one-vote-per-device-per-poll guard (unless the poll allows multi-voting).
	voterHash := hashVoter(c.ClientIP(), c.GetHeader("User-Agent"), pollID)
	if !poll.AllowMulti {
		count, err := h.Votes.CountDocuments(ctx, bson.M{"poll_id": objID, "voter_hash": voterHash})
		if err == nil && count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "you have already voted on this poll"})
			return
		}
	}

	vote := models.Vote{
		PollID:    objID,
		OptionID:  req.OptionID,
		VoterHash: voterHash,
		CreatedAt: time.Now().UTC(),
	}
	if _, err := h.Votes.InsertOne(ctx, vote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record vote"})
		return
	}

	// The live counter lives in Redis — this is what makes results update
	// instantly without hitting Mongo on every poll/refresh.
	newCount, err := h.Redis.HIncrBy(ctx, redisCountsKey(pollID), req.OptionID, 1).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update live counters"})
		return
	}
	_ = newCount

	results, err := h.currentResults(ctx, pollID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load live results"})
		return
	}

	// Publish so every connected WebSocket client (across this and any other
	// backend instance) gets the update — this is the "truly real-time" piece.
	payload, _ := json.Marshal(results)
	h.Redis.Publish(ctx, redisChannel(pollID), payload)

	// Also fan out directly through the in-process hub for clients on this instance.
	h.Hub.Broadcast(pollID, payload)

	c.JSON(http.StatusOK, results)
}

func (h *PollHandler) currentResults(ctx context.Context, pollID string) (models.PollResults, error) {
	raw, err := h.Redis.HGetAll(ctx, redisCountsKey(pollID)).Result()
	if err != nil {
		return models.PollResults{}, err
	}
	counts := make(map[string]int, len(raw))
	total := 0
	for k, v := range raw {
		n := 0
		fmt.Sscanf(v, "%d", &n)
		counts[k] = n
		total += n
	}
	return models.PollResults{PollID: pollID, Counts: counts, Total: total}, nil
}

func hashVoter(ip, ua, pollID string) string {
	h := sha256.Sum256([]byte(ip + "|" + ua + "|" + pollID))
	return hex.EncodeToString(h[:])
}
