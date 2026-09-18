package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/livepolling/backend/services"
)

// LiveHandler serves Server-Sent Events (SSE) for real-time vote updates.
type LiveHandler struct {
	polls    *mongo.Collection
	redisSvc *services.RedisService
}

// NewLiveHandler creates a new LiveHandler.
func NewLiveHandler(db *mongo.Database, redisSvc *services.RedisService) *LiveHandler {
	return &LiveHandler{
		polls:    db.Collection("polls"),
		redisSvc: redisSvc,
	}
}

// Stream opens an SSE connection that pushes live vote count updates.
//
// 1. Sends the current counts as an "init" event on connect.
// 2. Subscribes to the Redis Pub/Sub channel for this poll.
// 3. Forwards every vote update as a "vote" event.
// 4. Sends heartbeat comments every 15s to keep the connection alive.
func (h *LiveHandler) Stream(c *gin.Context) {
	pollID := c.Param("id")

	// Validate poll exists
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid poll ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = h.polls.FindOne(ctx, bson.M{"_id": objID}).Err()
	cancel()
	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}

	// --- SSE headers ---
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// --- Send initial state ---
	counts, _ := h.redisSvc.GetVoteCounts(context.Background(), pollID)
	if counts == nil {
		counts = make(map[string]int)
	}
	initialData, _ := json.Marshal(counts)
	fmt.Fprintf(c.Writer, "event: init\ndata: %s\n\n", initialData)
	c.Writer.Flush()

	// --- Subscribe to Redis Pub/Sub for this poll ---
	sub := h.redisSvc.Subscribe(context.Background(), pollID)
	defer sub.Close()

	ch := sub.Channel()
	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: vote\ndata: %s\n\n", msg.Payload)
			c.Writer.Flush()
		case <-ticker.C:
			// Heartbeat to detect dead connections
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}
