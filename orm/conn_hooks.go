package orm

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-kit/pool"
)

// ConnHooks 定义连接处理的钩子函数
type ConnHooks struct {
	// OnGet 在从连接池获取连接时调用
	OnGet func(ctx context.Context, conn *sql.DB) error

	// OnPut 在归还连接到连接池时调用
	OnPut func(conn *sql.DB, err error) error

	// OnCheckHealth 检查连接健康状态
	OnCheckHealth func(conn *sql.DB) bool

	// OnClose 在关闭连接时调用
	OnClose func(conn *sql.DB) error
}

// 默认的连接健康检查函数
func defaultHealthCheck(db *sql.DB) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	return db.PingContext(ctx) == nil
}

// ConnectionTracker 用于跟踪和管理连接的生命周期
type ConnectionTracker struct {
	mu            sync.Mutex
	activeConns   map[*sql.Rows]pool.Connection
	activeQueries map[*sql.Rows]bool
}

// NewConnectionTracker 创建一个新的连接跟踪器
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		activeConns:   make(map[*sql.Rows]pool.Connection),
		activeQueries: make(map[*sql.Rows]bool),
	}
}

// TrackRows 跟踪连接和查询结果集
func (ct *ConnectionTracker) TrackRows(rows *sql.Rows, conn pool.Connection) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.activeConns[rows] = conn
	ct.activeQueries[rows] = true
}

// ReleaseRows 释放连接并归还给池
func (ct *ConnectionTracker) ReleaseRows(rows *sql.Rows, pooledDB *PooledDB) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if conn, ok := ct.activeConns[rows]; ok {
		delete(ct.activeConns, rows)
		delete(ct.activeQueries, rows)
		pooledDB.PutConn(conn, nil)
	}
}

// WithConnHooks 配置连接钩子函数
func WithConnHooks(hooks *ConnHooks) DBOption {
	return func(db *DB) error {
		if db.pooledDB == nil {
			// 如果没有配置连接池，创建一个默认连接池
			pooledDB, err := NewPooledDB(db.sqlDB, DefaultDBPoolConfig())
			if err != nil {
				return err
			}
			db.pooledDB = pooledDB
		}

		// 注册钩子函数
		db.pooledDB.hooks = hooks
		return nil
	}
}

// nopCloser 不关闭连接的空实现
func nopCloser() func(*sql.DB) error {
	return func(db *sql.DB) error {
		return nil
	}
}

// withQueryTrackHook 创建跟踪查询的钩子
func withQueryTrackHook(tracker *ConnectionTracker, pooledDB *PooledDB) *ConnHooks {
	return &ConnHooks{
		OnGet: func(ctx context.Context, conn *sql.DB) error {
			// 跟踪获取连接时的相关信息
			return nil
		},
		OnPut: func(conn *sql.DB, err error) error {
			// 归还连接时的钩子逻辑
			return err
		},
		OnCheckHealth: defaultHealthCheck,
		OnClose:       nopCloser(),
	}
}