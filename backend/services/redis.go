package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/livepolling/backend/models"
	"github.com/redis/go-redis/v9"
)

// RedisService wraps Redis operations for vote counting, pub/sub, and dedup.
type RedisService struct {
	client *redis.Client
}

// NewRedisService connects to Redis using the provided URL.
func NewRedisService(redisURL string) (*RedisService, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisService{client: client}, nil
}

// InitializePoll sets up the Redis hash for vote counting when a poll is created.
func (s *RedisService) InitializePoll(ctx context.Context, pollID string, options []models.Option) error {
	key := fmt.Sprintf("poll:%s:votes", pollID)
	pipe := s.client.Pipeline()
	for _, opt := range options {
		pipe.HSet(ctx, key, opt.ID, 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// IncrementVote atomically increments the vote count for an option and returns all current counts.
func (s *RedisService) IncrementVote(ctx context.Context, pollID, optionID string) (map[string]int, error) {
	key := fmt.Sprintf("poll:%s:votes", pollID)
	if err := s.client.HIncrBy(ctx, key, optionID, 1).Err(); err != nil {
		return nil, err
	}
	return s.GetVoteCounts(ctx, pollID)
}

// GetVoteCounts retrieves all option vote counts from the Redis hash.
func (s *RedisService) GetVoteCounts(ctx context.Context, pollID string) (map[string]int, error) {
	key := fmt.Sprintf("poll:%s:votes", pollID)
	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for k, v := range result {
		count, _ := strconv.Atoi(v)
		counts[k] = count
	}
	return counts, nil
}

// CheckAndAddVoter uses a Redis SET to detect duplicate voters.
// Returns true if the voter has already voted (duplicate).
func (s *RedisService) CheckAndAddVoter(ctx context.Context, pollID, voterID string) (bool, error) {
	key := fmt.Sprintf("poll:%s:voters", pollID)
	added, err := s.client.SAdd(ctx, key, voterID).Result()
	if err != nil {
		return false, err
	}
	return added == 0, nil
}

// PublishVoteUpdate broadcasts updated vote counts to all SSE listeners via Redis Pub/Sub.
func (s *RedisService) PublishVoteUpdate(ctx context.Context, pollID string, counts map[string]int) error {
	channel := fmt.Sprintf("poll:%s", pollID)
	data, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, channel, data).Err()
}

// Subscribe returns a Redis Pub/Sub subscription for a specific poll channel.
func (s *RedisService) Subscribe(ctx context.Context, pollID string) *redis.PubSub {
	channel := fmt.Sprintf("poll:%s", pollID)
	return s.client.Subscribe(ctx, channel)
}

// CleanupPoll removes all Redis keys associated with a poll.
func (s *RedisService) CleanupPoll(ctx context.Context, pollID string) {
	pipe := s.client.Pipeline()
	pipe.Del(ctx, fmt.Sprintf("poll:%s:votes", pollID))
	pipe.Del(ctx, fmt.Sprintf("poll:%s:voters", pollID))
	pipe.Exec(ctx)
}

// Client returns the underlying Redis client for direct use (e.g., rate limiting).
func (s *RedisService) Client() *redis.Client {
	return s.client
}

// Close shuts down the Redis connection.
func (s *RedisService) Close() error {
	return s.client.Close()
}
