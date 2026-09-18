package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

type Option struct {
	ID   string `bson:"id" json:"id"` // short id, unique within the poll (e.g. "opt_1")
	Text string `bson:"text" json:"text"`
}

type Poll struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Question    string             `bson:"question" json:"question"`
	Options     []Option           `bson:"options" json:"options"`
	OwnerID     primitive.ObjectID `bson:"owner_id" json:"owner_id"`
	IsClosed    bool               `bson:"is_closed" json:"is_closed"`
	AllowMulti  bool               `bson:"allow_multi" json:"allow_multi"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// Vote is persisted for audit/history and duplicate-prevention.
// Live counts themselves are served from Redis, not computed by
// scanning this collection on every request.
type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    primitive.ObjectID `bson:"poll_id" json:"poll_id"`
	OptionID  string             `bson:"option_id" json:"option_id"`
	VoterHash string             `bson:"voter_hash" json:"-"` // hash of IP+UA+pollID, used to block duplicate votes
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// PollResults is what gets broadcast to WebSocket clients and returned by the API.
type PollResults struct {
	PollID string         `json:"poll_id"`
	Counts map[string]int `json:"counts"` // optionID -> vote count
	Total  int            `json:"total"`
}
