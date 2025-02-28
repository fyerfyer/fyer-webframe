package redissession

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-webframe/web/session"
	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	client          redis.Cmdable
	expireTime      time.Duration
	prefix          string
	sessions        sync.Map // 添加session缓存池
	cleanupInterval time.Duration
	stopCleanup     context.CancelFunc
}

var defaultExpireTime = time.Duration(3600) * time.Second
var defaultPrefix = "sess_"
var defaultCleanupInterval = 5 * time.Minute

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

func WithCleanupInterval(interval time.Duration) RedisStorageOption {
	return func(rs *RedisStorage) {
		rs.cleanupInterval = interval
	}
}

func NewRedisStorage(redis redis.Cmdable, opts ...RedisStorageOption) *RedisStorage {
	ctx, cancel := context.WithCancel(context.Background())

	rs := &RedisStorage{
		client:          redis,
		expireTime:      defaultExpireTime,
		prefix:          defaultPrefix,
		cleanupInterval: defaultCleanupInterval,
		stopCleanup:     cancel,
	}

	for _, opt := range opts {
		opt(rs)
	}

	// 启动清理goroutine
	go rs.startCleanup(ctx)

	return rs
}

func (r *RedisStorage) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.GC(ctx); err != nil {
				// 记录错误日志
				log.Fatalf("error cleaning up expired sessions: %v", err)
			}
		}
	}
}

// GC 从session缓存池中清除过期的session
func (r *RedisStorage) GC(ctx context.Context) error {
	// Clean local cache based on Redis data
	r.sessions.Range(func(key, value interface{}) bool {
		id := key.(string)
		exists, err := r.client.Exists(ctx, r.prefix+id).Result()
		if err != nil || exists == 0 {
			r.sessions.Delete(id)
		}
		return true
	})

	return nil
}

// Close 关闭清理goroutine
func (r *RedisStorage) Close() error {
	if r.stopCleanup != nil {
		r.stopCleanup()
	}
	return nil
}

// Create 创建并返回session
func (r *RedisStorage) Create(ctx context.Context, id string) (session.Session, error) {
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
		expiration:   r.expireTime,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, nil
}

// Find 查询并返回session
func (r *RedisStorage) Find(ctx context.Context, id string) (session.Session, error) {
	// 先从缓存池中查找
	if sessVal, ok := r.sessions.Load(id); ok {
		sess := sessVal.(*Session)
		// 检查 Redis 中的会话是否仍然存在（可能已过期）
        exists, err := r.client.Exists(ctx, r.prefix+id).Result()
        if err != nil {
            return nil, err
        }
        if exists == 0 {
            // 会话已过期或被删除，从本地缓存中移除
            r.sessions.Delete(id)
            return nil, redis.Nil
        }
        return sess, nil
	}

	// 检查session是否存在，以保持数据一致性
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
		expiration:   r.expireTime,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, nil
}

// Refresh 刷新session的过期时间
func (r *RedisStorage) Refresh(ctx context.Context, id string) error {
	_, err := r.client.Expire(ctx, r.prefix+id, r.expireTime).Result()
	return err
}

// Delete 删除session
func (r *RedisStorage) Delete(ctx context.Context, id string) error {
	// 从缓存池中删除
	r.sessions.Delete(id)

	_, err := r.client.Del(ctx, r.prefix+id).Result()
	return err
}
