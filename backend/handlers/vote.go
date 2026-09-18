package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/livepolling/backend/models"
	"github.com/livepolling/backend/services"
)

// VoteHandler processes incoming votes with validation, dedup, and live broadcasting.
type VoteHandler struct {
	polls    *mongo.Collection
	votes    *mongo.Collection
	redisSvc *services.RedisService
}

// NewVoteHandler creates a new VoteHandler.
func NewVoteHandler(db *mongo.Database, redisSvc *services.RedisService) *VoteHandler {
	return &VoteHandler{
		polls:    db.Collection("polls"),
		votes:    db.Collection("votes"),
		redisSvc: redisSvc,
	}
}

type voteRequest struct {
	OptionID string `json:"option_id" binding:"required"`
}

// Vote validates the vote, checks for duplicates, increments atomically in Redis,
// publishes the update via Pub/Sub, and stores a record in MongoDB.
func (h *VoteHandler) Vote(c *gin.Context) {
	pollID := c.Param("id")

	var req voteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "option_id is required"})
		return
	}

	if len(req.OptionID) == 0 || len(req.OptionID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option ID"})
		return
	}

	// --- Validate poll exists ---
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	if err := h.polls.FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// --- Check poll is active ---
	if !poll.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "This poll is closed"})
		return
	}
	if poll.ExpiresAt != nil && time.Now().After(*poll.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This poll has expired"})
		return
	}

	// --- Validate option exists ---
	optionValid := false
	for _, opt := range poll.Options {
		if opt.ID == req.OptionID {
			optionValid = true
			break
		}
	}
	if !optionValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid option"})
		return
	}

	// --- Duplicate vote detection (Redis SET) ---
	voterID := generateVoterID(c)
	isDuplicate, err := h.redisSvc.CheckAndAddVoter(context.Background(), pollID, voterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process vote"})
		return
	}
	if isDuplicate {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already voted on this poll"})
		return
	}

	// --- Atomic increment in Redis ---
	newCounts, err := h.redisSvc.IncrementVote(context.Background(), pollID, req.OptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote"})
		return
	}

	// --- Persist vote record to MongoDB (async) ---
	vote := models.Vote{
		PollID:          objID,
		OptionID:        req.OptionID,
		VoterIdentifier: voterID,
		CreatedAt:       time.Now(),
	}
	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer bgCancel()
		if _, err := h.votes.InsertOne(bgCtx, vote); err != nil {
			log.Printf("Failed to persist vote for poll %s: %v", pollID, err)
		}
	}()

	// --- Broadcast update via Redis Pub/Sub ---
	h.redisSvc.PublishVoteUpdate(context.Background(), pollID, newCounts)

	c.JSON(http.StatusOK, gin.H{
		"message": "Vote recorded",
		"counts":  newCounts,
	})
}

// generateVoterID creates a SHA-256 hash from IP + User-Agent for fingerprinting.
func generateVoterID(c *gin.Context) string {
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", ip, ua)))
	return hex.EncodeToString(hash[:])
}
