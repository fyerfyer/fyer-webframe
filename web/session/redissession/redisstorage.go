package redissession

import (
	"context"
	"github.com/fyerfyer/fyer-webframe/web/session"
	"log"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-kit/pool"
	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	redisPool       pool.Pool          // 使用连接池代替直接的Redis客户端
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

// NewRedisStorage 创建一个新的Redis会话存储，使用连接池
func NewRedisStorage(redisPool pool.Pool, opts ...RedisStorageOption) *RedisStorage {
	ctx, cancel := context.WithCancel(context.Background())

	rs := &RedisStorage{
		redisPool:       redisPool,
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
				log.Printf("error cleaning up expired sessions: %v", err)
			}
		}
	}
}

// GC 从session缓存池中清除过期的session
func (r *RedisStorage) GC(ctx context.Context) error {
	// 从连接池获取一个连接
	conn, err := r.redisPool.Get(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 获取底层Redis客户端
	client, ok := conn.Raw().(*redis.Client)
	if !ok {
		return r.redisPool.Put(conn, err)
	}

	// Clean local cache based on Redis data
	r.sessions.Range(func(key, value interface{}) bool {
		id := key.(string)
		exists, err := client.Exists(ctx, r.prefix+id).Result()
		if err != nil || exists == 0 {
			r.sessions.Delete(id)
		}
		return true
	})

	return r.redisPool.Put(conn, nil)
}

// Close 关闭清理goroutine和连接池
func (r *RedisStorage) Close() error {
	if r.stopCleanup != nil {
		r.stopCleanup()
	}
	return nil
}

// Create 创建并返回session
func (r *RedisStorage) Create(ctx context.Context, id string) (session.Session, error) {
	// 从连接池获取一个连接
	conn, err := r.redisPool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 获取底层Redis客户端
	client, ok := conn.Raw().(*redis.Client)
	if !ok {
		return nil, r.redisPool.Put(conn, err)
	}

	key := r.prefix + id

	// 创建session hash并设置过期时间
	err = client.HSet(ctx, key, "_created", time.Now().Unix()).Err()
	if err != nil {
		return nil, r.redisPool.Put(conn, err)
	}

	err = client.Expire(ctx, key, r.expireTime).Err()
	if err != nil {
		return nil, r.redisPool.Put(conn, err)
	}

	// 创建新的会话，使用连接池
	sess := &Session{
		id:           id,
		data:         make(map[string]any),
		redisPool:    r.redisPool, // 使用连接池代替直接的Redis客户端
		prefix:       r.prefix,
		expiration:   r.expireTime,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, r.redisPool.Put(conn, nil)
}

// Find 查询并返回session
func (r *RedisStorage) Find(ctx context.Context, id string) (session.Session, error) {
	// 先从缓存池中查找
	if sessVal, ok := r.sessions.Load(id); ok {
		sess := sessVal.(*Session)

		// 从连接池获取一个连接来验证会话是否仍然存在
		conn, err := r.redisPool.Get(ctx)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		// 获取底层Redis客户端
		client, ok := conn.Raw().(*redis.Client)
		if !ok {
			return nil, r.redisPool.Put(conn, err)
		}

		// 检查 Redis 中的会话是否仍然存在（可能已过期）
		exists, err := client.Exists(ctx, r.prefix+id).Result()
		if err != nil {
			return nil, r.redisPool.Put(conn, err)
		}

		if exists == 0 {
			// 会话已过期或被删除，从本地缓存中移除
			r.sessions.Delete(id)
			r.redisPool.Put(conn, nil)
			return nil, redis.Nil
		}

		r.redisPool.Put(conn, nil)
		return sess, nil
	}

	// 如果缓存中没有，从Redis获取
	conn, err := r.redisPool.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 获取底层Redis客户端
	client, ok := conn.Raw().(*redis.Client)
	if !ok {
		return nil, r.redisPool.Put(conn, err)
	}

	// 检查session是否存在
	exists, err := client.Exists(ctx, r.prefix+id).Result()
	if err != nil {
		return nil, r.redisPool.Put(conn, err)
	}

	if exists == 0 {
		r.redisPool.Put(conn, nil)
		return nil, redis.Nil
	}

	// 创建新的会话，使用连接池
	sess := &Session{
		id:           id,
		data:         make(map[string]any),
		redisPool:    r.redisPool, // 使用连接池代替直接的Redis客户端
		prefix:       r.prefix,
		expiration:   r.expireTime,
	}

	// 将session存入缓存池
	r.sessions.Store(id, sess)

	return sess, r.redisPool.Put(conn, nil)
}

// Refresh 刷新session的过期时间
func (r *RedisStorage) Refresh(ctx context.Context, id string) error {
	conn, err := r.redisPool.Get(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 获取底层Redis客户端
	client, ok := conn.Raw().(*redis.Client)
	if !ok {
		return r.redisPool.Put(conn, err)
	}

	_, err = client.Expire(ctx, r.prefix+id, r.expireTime).Result()
	return r.redisPool.Put(conn, err)
}

// Delete 删除session
func (r *RedisStorage) Delete(ctx context.Context, id string) error {
	// 从缓存池中删除
	r.sessions.Delete(id)

	conn, err := r.redisPool.Get(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 获取底层Redis客户端
	client, ok := conn.Raw().(*redis.Client)
	if !ok {
		return r.redisPool.Put(conn, err)
	}

	_, err = client.Del(ctx, r.prefix+id).Result()
	return r.redisPool.Put(conn, err)
}