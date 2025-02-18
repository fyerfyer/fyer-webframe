package redissession

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sync"
)

type Session struct {
	id           string
	data         map[string]any
	sessionRedis redis.Cmdable
	prefix       string
	mu           sync.RWMutex // 添加读写锁
}

func (s *Session) Get(ctx context.Context, key string) (any, error) {
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

func (s *Session) Set(ctx context.Context, key string, value any) error {
	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %v", err)
	}

	// 保存到Redis
	err = s.sessionRedis.HSet(context.Background(),
		s.prefix+s.id,
		key,
		string(data),
	).Err()
	if err != nil {
		return fmt.Errorf("failed to save to redis: %v", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	return nil
}

func (s *Session) ID() string {
	return s.id
}
