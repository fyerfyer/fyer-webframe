package redissession

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type Session struct {
	id           string
	data         map[string]any
	sessionRedis redis.Cmdable
	prefix       string
	mu           sync.RWMutex // 添加读写锁
	expiration   time.Duration
}

// Get 获取session中的值
func (s *Session) Get(ctx context.Context, key string) (any, error) {
	// 在获取值前先刷新session，以防止session过期
	if s.expiration > 0 {
		err := s.Touch(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh session: %w", err)
		}
	}

	s.mu.RLock() // 读取加读锁
	if val, ok := s.data[key]; ok {
		s.mu.RUnlock()
		return val, nil
	}
	s.mu.RUnlock()

	// 从Redis获取
	val, err := s.sessionRedis.HGet(ctx, s.prefix+s.id, key).Result()
	if err != nil {
		return nil, err
	}

	// 反序列化
	var result any
	err = json.Unmarshal([]byte(val), &result)
	if err != nil {
		return nil, err
	}

	// 更新本地缓存
	s.mu.Lock() // 写入加写锁
	s.data[key] = result
	s.mu.Unlock()

	return result, nil
}

// Set 设置session中的值
func (s *Session) Set(ctx context.Context, key string, value any) error {
	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %v", err)
	}

	// 保存到Redis
	err = s.sessionRedis.HSet(ctx, s.prefix+s.id, key, string(data)).Err()
	if err != nil {
		return fmt.Errorf("failed to save to redis: %v", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	// 在写入时刷新session
	if s.expiration > 0 {
		if err := s.Touch(ctx); err != nil {
			return fmt.Errorf("failed to refresh session: %w", err)
		}
	}

	return nil
}

// ID 返回session ID
func (s *Session) ID() string {
	return s.id
}

// Touch 更新session的过期时间
func (s *Session) Touch(ctx context.Context) error {
	if s.expiration <= 0 {
		return nil
	}

	// 更新过期时间
	return s.sessionRedis.Expire(ctx, s.prefix+s.id, s.expiration).Err()
}
