package redissession

import (
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

type RedisStorage struct {
	client     redis.Cmdable
	expireTime time.Duration
	prefix     string
	sessions   sync.Map // 添加session缓存池
}

var defaultExpireTime = time.Duration(3600) * time.Second
var defaultPrefix = "sess_"

type RedisStorageOption func(*RedisStorage)

func WithExpireTime(expireTime time.Duration) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.expireTime = expireTime
	}
}

func WithPrefix(prefix string) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.prefix = prefix
	}
}

func NewRedisStorage(redis redis.Cmdable) *RedisStorage {
	return &RedisStorage{
		client:     redis,
		expireTime: defaultExpireTime,
		prefix:     defaultPrefix,
	}
}

func (r *RedisStorage) Create(ctx context.Context, id string) (*Session, error) {
	key := r.prefix + id

	// 创建session hash并设置过期时间
	err := r.client.HSet(ctx, key, "_created", time.Now().Unix()).Err()
	if err != nil {
		return nil, err
	}

	err = r.client.Expire(ctx, key, r.expireTime).Err()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		id:           id,
		data:         make(map[string]any),
		sessionRedis: r.client,
		prefix:       r.prefix,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, nil
}

func (r *RedisStorage) Find(ctx context.Context, id string) (*Session, error) {
	// 先从缓存池中查找
	if sess, ok := r.sessions.Load(id); ok {
		return sess.(*Session), nil
	}

	// 检查session是否存在
	exists, err := r.client.Exists(ctx, r.prefix+id).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, redis.Nil
	}

	sess := &Session{
		id:           id,
		data:         make(map[string]any),
		sessionRedis: r.client,
		prefix:       r.prefix,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, nil
}

func (r RedisStorage) Refresh(ctx context.Context, id string) error {
	_, err := r.client.Expire(ctx, r.prefix+id, r.expireTime).Result()
	return err
}

func (r *RedisStorage) Delete(ctx context.Context, id string) error {
	// 从缓存池中删除
	r.sessions.Delete(id)

	_, err := r.client.Del(ctx, r.prefix+id).Result()
	return err
}
