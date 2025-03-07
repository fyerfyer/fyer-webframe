package orm

import (
	"database/sql"
	"time"

	"github.com/fyerfyer/fyer-kit/pool"
)

// DBPoolConfig 定义数据库连接池配置
type DBPoolConfig struct {
	// 连接池配置
	MaxIdle     int           // 最大空闲连接数
	MaxActive   int           // 最大活动连接数(0表示无限制)
	MaxIdleTime time.Duration // 连接最大空闲时间
	MaxLifetime time.Duration // 连接最大生命周期
	InitialSize int           // 初始连接数
	WaitTimeout time.Duration // 等待可用连接的超时时间
	DialTimeout time.Duration // 连接超时时间

	// 健康检查
	HealthCheck func(*sql.DB) bool
}

// DefaultDBPoolConfig 返回默认的连接池配置
func DefaultDBPoolConfig() *DBPoolConfig {
	return &DBPoolConfig{
		MaxIdle:     10,
		MaxActive:   100,
		MaxIdleTime: 5 * time.Minute,
		MaxLifetime: 30 * time.Minute,
		InitialSize: 5,
		WaitTimeout: 3 * time.Second,
		DialTimeout: 2 * time.Second,
		HealthCheck: defaultHealthCheck,
	}
}

// WithPool 启用连接池
func WithPool(config *DBPoolConfig) DBOption {
	return func(db *DB) error {
		pooled, err := NewPooledDB(db.sqlDB, config)
		if err != nil {
			return err
		}
		db.pooledDB = pooled
		return nil
	}
}

// WithExistingPool 使用已存在的连接池
func WithExistingPool(p pool.Pool) DBOption {
	return func(db *DB) error {
		db.pooledDB = &PooledDB{
			sqlDB:  db.sqlDB,
			pool:   p,
			pooled: true,
		}
		return nil
	}
}

// WithConnectionPool 配置连接池
func WithConnectionPool(opts ...DBPoolOption) DBOption {
	return func(db *DB) error {
		config := DefaultDBPoolConfig()
		for _, opt := range opts {
			opt(config)
		}
		return WithPool(config)(db)
	}
}

// DBPoolOption 定义连接池配置选项
type DBPoolOption func(*DBPoolConfig)

// WithPoolMaxIdle 设置最大空闲连接数
func WithPoolMaxIdle(maxIdle int) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.MaxIdle = maxIdle
	}
}

// WithPoolMaxActive 设置最大活动连接数
func WithPoolMaxActive(maxActive int) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.MaxActive = maxActive
	}
}

// WithPoolMaxIdleTime 设置连接最大空闲时间
func WithPoolMaxIdleTime(duration time.Duration) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.MaxIdleTime = duration
	}
}

// WithPoolMaxLifetime 设置连接最大生命周期
func WithPoolMaxLifetime(duration time.Duration) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.MaxLifetime = duration
	}
}

// WithPoolInitialSize 设置初始连接数
func WithPoolInitialSize(size int) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.InitialSize = size
	}
}

// WithPoolWaitTimeout 设置等待连接超时时间
func WithPoolWaitTimeout(duration time.Duration) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.WaitTimeout = duration
	}
}

// WithPoolDialTimeout 设置连接超时时间
func WithPoolDialTimeout(duration time.Duration) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.DialTimeout = duration
	}
}

// WithPoolHealthCheck 设置健康检查函数
func WithPoolHealthCheck(check func(*sql.DB) bool) DBPoolOption {
	return func(config *DBPoolConfig) {
		config.HealthCheck = check
	}
}

// 便捷方法，直接配置一些常用的连接池参数
func WithPoolSize(maxIdle, maxActive int) DBOption {
	return WithConnectionPool(
		WithPoolMaxIdle(maxIdle),
		WithPoolMaxActive(maxActive),
	)
}

// WithPoolTimeouts 设置连接池超时参数
func WithPoolTimeouts(idle, lifetime time.Duration) DBOption {
	return WithConnectionPool(
		WithPoolMaxIdleTime(idle),
		WithPoolMaxLifetime(lifetime),
	)
}
