package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SyncService periodically flushes Redis vote counts to MongoDB for persistence.
type SyncService struct {
	polls    *mongo.Collection
	redisSvc *RedisService
	stopCh   chan struct{}
}

// NewSyncService creates a new sync worker.
func NewSyncService(db *mongo.Database, redisSvc *RedisService) *SyncService {
	return &SyncService{
		polls:    db.Collection("polls"),
		redisSvc: redisSvc,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the background sync loop (every 30 seconds).
func (s *SyncService) Start() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.syncVotesToMongo()
			case <-s.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
	log.Println("✓ Sync service started (Redis → MongoDB every 30s)")
}

// Stop gracefully shuts down the sync worker.
func (s *SyncService) Stop() {
	close(s.stopCh)
}

func (s *SyncService) syncVotesToMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := s.polls.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		log.Printf("Sync error (find polls): %v", err)
		return
	}
	defer cursor.Close(ctx)

	type pollDoc struct {
		ID      primitive.ObjectID `bson:"_id"`
		Options []struct {
			ID string `bson:"id"`
		} `bson:"options"`
	}

	synced := 0
	for cursor.Next(ctx) {
		var p pollDoc
		if err := cursor.Decode(&p); err != nil {
			continue
		}

		counts, err := s.redisSvc.GetVoteCounts(ctx, p.ID.Hex())
		if err != nil || len(counts) == 0 {
			continue
		}

		update := bson.M{}
		total := 0
		for i, opt := range p.Options {
			if count, ok := counts[opt.ID]; ok {
				key := fmt.Sprintf("options.%d.vote_count", i)
				update[key] = count
				total += count
			}
		}
		update["total_votes"] = total

		_, err = s.polls.UpdateOne(ctx,
			bson.M{"_id": p.ID},
			bson.M{"$set": update},
		)
		if err != nil {
			log.Printf("Sync error (update poll %s): %v", p.ID.Hex(), err)
		} else {
			synced++
		}
	}

	if synced > 0 {
		log.Printf("Synced %d poll(s) from Redis → MongoDB", synced)
	}
}
