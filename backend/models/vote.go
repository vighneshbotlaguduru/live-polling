package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Vote records a single vote cast by a voter on a poll option.
type Vote struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID          primitive.ObjectID `bson:"poll_id" json:"poll_id"`
	OptionID        string             `bson:"option_id" json:"option_id"`
	VoterIdentifier string             `bson:"voter_identifier" json:"voter_identifier"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}
