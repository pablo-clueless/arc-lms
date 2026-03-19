package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// KeyTTL is the time-to-live for idempotency keys (24 hours)
	KeyTTL = 24 * time.Hour
	// KeyPrefix is the prefix for idempotency keys in Redis
	KeyPrefix = "idempotency"
)

// StoredResponse represents a stored API response for idempotency
type StoredResponse struct {
	StatusCode   int                    `json:"status_code"`
	ResponseBody map[string]interface{} `json:"response_body"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Store handles idempotency key storage in Redis
type Store struct {
	client *redis.Client
}

// NewStore creates a new idempotency store
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// buildKey builds a Redis key for an idempotency key
func (s *Store) buildKey(tenantID uuid.UUID, idempotencyKey string) string {
	return fmt.Sprintf("%s:%s:%s", KeyPrefix, tenantID.String(), idempotencyKey)
}

// TryAcquire attempts to acquire an idempotency key
// Returns true if acquired (first time), false if already exists
func (s *Store) TryAcquire(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (bool, error) {
	key := s.buildKey(tenantID, idempotencyKey)

	// Use SET NX (set if not exists) with expiry
	// Store a placeholder value initially
	success, err := s.client.SetNX(ctx, key, "processing", KeyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire idempotency key: %w", err)
	}

	return success, nil
}

// Get retrieves a stored response for an idempotency key
func (s *Store) Get(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) (*StoredResponse, error) {
	key := s.buildKey(tenantID, idempotencyKey)

	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Key doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}

	// If still processing, return nil
	if data == "processing" {
		return nil, nil
	}

	var response StoredResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stored response: %w", err)
	}

	return &response, nil
}

// Store stores a response for an idempotency key
func (s *Store) Store(ctx context.Context, tenantID uuid.UUID, idempotencyKey string, statusCode int, responseBody map[string]interface{}) error {
	key := s.buildKey(tenantID, idempotencyKey)

	response := StoredResponse{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Store with TTL
	if err := s.client.Set(ctx, key, data, KeyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store response: %w", err)
	}

	return nil
}

// Delete removes an idempotency key (used in case of errors)
func (s *Store) Delete(ctx context.Context, tenantID uuid.UUID, idempotencyKey string) error {
	key := s.buildKey(tenantID, idempotencyKey)
	return s.client.Del(ctx, key).Err()
}
