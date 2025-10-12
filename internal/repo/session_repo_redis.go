package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type sessionRepoRedis struct {
	rdb *redis.Client
}

func NewSessionRepoRedis(rdb *redis.Client) SessionRepo {
	return &sessionRepoRedis{rdb: rdb}
}

type sessionValue struct {
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"` // RFC3339
}

func (s *sessionRepoRedis) key(tok string) string { return "sess:" + tok }

func (s *sessionRepoRedis) Create(token string, userID string, expiresRFC3339 string) error {
	// compute TTL; also store payload for explicit expiry checks
	exp, err := time.Parse(time.RFC3339, expiresRFC3339)
	if err != nil {
		return err
	}
	ttl := time.Until(exp)
	if ttl < 0 {
		ttl = 0
	}
	val, _ := json.Marshal(sessionValue{UserID: userID, ExpiresAt: exp.UTC().Format(time.RFC3339)})
	return s.rdb.Set(context.Background(), s.key(token), val, ttl).Err()
}

func (s *sessionRepoRedis) Delete(token string) error {
	return s.rdb.Del(context.Background(), s.key(token)).Err()
}

func (s *sessionRepoRedis) Lookup(token string) (string, string, error) {
	v, err := s.rdb.Get(context.Background(), s.key(token)).Result()
	if err != nil {
		return "", "", err
	}
	var sv sessionValue
	if err := json.Unmarshal([]byte(v), &sv); err != nil {
		return "", "", err
	}
	return sv.UserID, sv.ExpiresAt, nil
}
