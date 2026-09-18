package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Option represents a single selectable choice within a poll.
type Option struct {
	ID        string `bson:"id" json:"id"`
	Text      string `bson:"text" json:"text"`
	VoteCount int    `bson:"vote_count" json:"vote_count"`
}

// Poll represents a live poll with its question, options, and metadata.
type Poll struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatorID  primitive.ObjectID `bson:"creator_id" json:"creator_id"`
	Question   string             `bson:"question" json:"question"`
	Options    []Option           `bson:"options" json:"options"`
	ShareCode  string             `bson:"share_code" json:"share_code"`
	IsActive   bool               `bson:"is_active" json:"is_active"`
	ExpiresAt  *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	TotalVotes int                `bson:"total_votes" json:"total_votes"`
}
