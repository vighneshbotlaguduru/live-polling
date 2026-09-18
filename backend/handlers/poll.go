package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/livepolling/backend/models"
	"github.com/livepolling/backend/services"
)

// PollHandler manages poll CRUD operations.
type PollHandler struct {
	polls    *mongo.Collection
	redisSvc *services.RedisService
}

// NewPollHandler creates a new PollHandler.
func NewPollHandler(db *mongo.Database, redisSvc *services.RedisService) *PollHandler {
	return &PollHandler{
		polls:    db.Collection("polls"),
		redisSvc: redisSvc,
	}
}

type createPollRequest struct {
	Question  string   `json:"question" binding:"required"`
	Options   []string `json:"options" binding:"required"`
	ExpiresIn *int     `json:"expires_in"` // duration in minutes; nil = no expiration
}

// Create validates input and stores a new poll in MongoDB + initializes Redis counters.
func (h *PollHandler) Create(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req createPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Question and options are required"})
		return
	}

	// --- Validate question ---
	req.Question = strings.TrimSpace(req.Question)
	if len(req.Question) < 5 || len(req.Question) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Question must be 5–500 characters"})
		return
	}

	// --- Validate options ---
	if len(req.Options) < 2 || len(req.Options) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polls require 2–10 options"})
		return
	}

	pollOptions := make([]models.Option, 0, len(req.Options))
	seen := make(map[string]bool)
	for _, text := range req.Options {
		text = strings.TrimSpace(text)
		if len(text) == 0 || len(text) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Each option must be 1–200 characters"})
			return
		}
		lower := strings.ToLower(text)
		if seen[lower] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate options are not allowed"})
			return
		}
		seen[lower] = true
		pollOptions = append(pollOptions, models.Option{
			ID:        uuid.New().String()[:8],
			Text:      text,
			VoteCount: 0,
		})
	}

	creatorObjID, _ := primitive.ObjectIDFromHex(userID.(string))
	shareCode := generateShareCode()

	poll := models.Poll{
		CreatorID:  creatorObjID,
		Question:   req.Question,
		Options:    pollOptions,
		ShareCode:  shareCode,
		IsActive:   true,
		CreatedAt:  time.Now(),
		TotalVotes: 0,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Minute)
		poll.ExpiresAt = &exp
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.polls.InsertOne(ctx, poll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create poll"})
		return
	}

	poll.ID = result.InsertedID.(primitive.ObjectID)

	// Initialize vote counters in Redis
	h.redisSvc.InitializePoll(context.Background(), poll.ID.Hex(), pollOptions)

	c.JSON(http.StatusCreated, poll)
}

// List returns all polls owned by the authenticated user, enriched with live Redis counts.
func (h *PollHandler) List(c *gin.Context) {
	userID, _ := c.Get("userID")
	creatorObjID, _ := primitive.ObjectIDFromHex(userID.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.polls.Find(ctx, bson.M{"creator_id": creatorObjID}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch polls"})
		return
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode polls"})
		return
	}
	if polls == nil {
		polls = []models.Poll{}
	}

	// Enrich with real-time vote counts from Redis
	for i := range polls {
		h.enrichWithRedis(&polls[i])
	}

	c.JSON(http.StatusOK, polls)
}

// Get retrieves a single poll by ObjectID or share code (public endpoint).
func (h *PollHandler) Get(c *gin.Context) {
	identifier := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var poll models.Poll
	var err error

	// Try as ObjectID first, then as share code
	objID, parseErr := primitive.ObjectIDFromHex(identifier)
	if parseErr == nil {
		err = h.polls.FindOne(ctx, bson.M{"_id": objID}).Decode(&poll)
	} else {
		err = h.polls.FindOne(ctx, bson.M{"share_code": identifier}).Decode(&poll)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// Auto-close expired polls (persist the state change)
	if poll.ExpiresAt != nil && time.Now().After(*poll.ExpiresAt) && poll.IsActive {
		poll.IsActive = false
		h.polls.UpdateOne(ctx, bson.M{"_id": poll.ID}, bson.M{"$set": bson.M{"is_active": false}})
	}

	h.enrichWithRedis(&poll)
	c.JSON(http.StatusOK, poll)
}

// Close marks a poll as inactive (owner only).
func (h *PollHandler) Close(c *gin.Context) {
	userID, _ := c.Get("userID")
	pollID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID"})
		return
	}

	creatorObjID, _ := primitive.ObjectIDFromHex(userID.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.polls.UpdateOne(ctx,
		bson.M{"_id": objID, "creator_id": creatorObjID},
		bson.M{"$set": bson.M{"is_active": false}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close poll"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or you are not the owner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poll closed successfully"})
}

// Delete removes a poll and its Redis data (owner only).
func (h *PollHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("userID")
	pollID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID"})
		return
	}

	creatorObjID, _ := primitive.ObjectIDFromHex(userID.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := h.polls.DeleteOne(ctx, bson.M{"_id": objID, "creator_id": creatorObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete poll"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found or you are not the owner"})
		return
	}

	h.redisSvc.CleanupPoll(context.Background(), pollID)
	c.JSON(http.StatusOK, gin.H{"message": "Poll deleted successfully"})
}

// enrichWithRedis overlays live Redis vote counts onto a poll struct.
func (h *PollHandler) enrichWithRedis(poll *models.Poll) {
	counts, err := h.redisSvc.GetVoteCounts(context.Background(), poll.ID.Hex())
	if err != nil || len(counts) == 0 {
		return
	}
	total := 0
	for i := range poll.Options {
		if count, ok := counts[poll.Options[i].ID]; ok {
			poll.Options[i].VoteCount = count
			total += count
		}
	}
	poll.TotalVotes = total
}

func generateShareCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
