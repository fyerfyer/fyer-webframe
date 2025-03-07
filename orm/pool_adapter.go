package orm

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-kit/pool"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

// SQLConnection 是对pool.Connection的封装，特定于ORM的SQL连接包装器
type SQLConnection struct {
	conn    pool.Connection // 底层连接池连接
	sqlConn *sql.DB         // 实际的SQL连接
	lastUse time.Time       // 最后使用时间
	mu      sync.RWMutex    // 保护并发访问
	closed  bool            // 是否已关闭
}

// Close 实现pool.Connection接口
func (c *SQLConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return nil // 实际的关闭由连接池处理
}

// Raw 返回底层SQL连接
func (c *SQLConnection) Raw() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sqlConn
}

// IsAlive 检查连接是否有效
func (c *SQLConnection) IsAlive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false
	}

	// 尝试ping来验证连接是否有效
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return c.sqlConn.PingContext(ctx) == nil
}

// ResetState 重置连接状态
func (c *SQLConnection) ResetState() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastUse = time.Now()
	c.closed = false
	return nil
}

// SQLConnectionFactory 用于创建新的SQL连接
type SQLConnectionFactory struct {
	sqlDB       *sql.DB
	healthCheck func(*sql.DB) bool
}

// NewSQLConnectionFactory 创建一个新的SQL连接工厂
func NewSQLConnectionFactory(db *sql.DB) *SQLConnectionFactory {
	return &SQLConnectionFactory{
		sqlDB:       db,
		healthCheck: defaultHealthCheck,
	}
}

// Create 实现ConnectionFactory接口，创建一个新的连接
func (f *SQLConnectionFactory) Create(ctx context.Context) (pool.Connection, error) {
	if f.sqlDB == nil {
		return nil, ferr.ErrCreateConnectionFailed(ferr.ErrDBClosed)
	}

	// 检查连接是否有效
	if !f.healthCheck(f.sqlDB) {
		return nil, ferr.ErrCreateConnectionFailed(ferr.ErrInvalidConnection)
	}

	return &SQLConnection{
		sqlConn: f.sqlDB,
		lastUse: time.Now(),
	}, nil
}

// WithHealthCheck 设置自定义健康检查函数
func (f *SQLConnectionFactory) WithHealthCheck(check func(*sql.DB) bool) *SQLConnectionFactory {
	f.healthCheck = check
	return f
}

// PooledDB 是支持连接池的数据库
type PooledDB struct {
	sqlDB      *sql.DB       // 原始数据库连接，非池化模式使用
	pool       pool.Pool     // 连接池
	pooled     bool          // 是否启用连接池
	poolConfig *DBPoolConfig // 连接池配置
	hooks      *ConnHooks    // 连接钩子函数
}

// NewPooledDB 创建一个新的池化数据库
func NewPooledDB(sqlDB *sql.DB, config *DBPoolConfig) (*PooledDB, error) {
	if config == nil {
		// 不使用连接池
		return &PooledDB{
			sqlDB:  sqlDB,
			pooled: false,
		}, nil
	}

	// 创建连接工厂
	factory := NewSQLConnectionFactory(sqlDB)
	if config.HealthCheck != nil {
		factory.WithHealthCheck(config.HealthCheck)
	}

	// 构建连接池选项
	options := []pool.Option{
		pool.WithMaxIdle(config.MaxIdle),
		pool.WithMaxActive(config.MaxActive),
		pool.WithMaxIdleTime(config.MaxIdleTime),
		pool.WithMaxLifetime(config.MaxLifetime),
		pool.WithWaitTimeout(config.WaitTimeout),
		pool.WithDialTimeout(config.DialTimeout),
		pool.WithInitialSize(config.InitialSize),
	}

	// 创建连接池
	p := pool.NewPool(factory, options...)

	return &PooledDB{
		sqlDB:      sqlDB,
		pool:       p,
		pooled:     true,
		poolConfig: config,
	}, nil
}

// GetConn 从池中获取一个连接，并应用获取钩子
func (pdb *PooledDB) GetConn(ctx context.Context) (*sql.DB, pool.Connection, error) {
	if !pdb.pooled {
		return pdb.sqlDB, nil, nil
	}

	conn, err := pdb.pool.Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	sqlConn, ok := conn.Raw().(*sql.DB)
	if !ok {
		pdb.pool.Put(conn, ferr.ErrInvalidConnection)
		return nil, nil, ferr.ErrInvalidConnection
	}

	// 应用健康检查钩子
	if pdb.hooks != nil && pdb.hooks.OnCheckHealth != nil {
		if !pdb.hooks.OnCheckHealth(sqlConn) {
			pdb.pool.Put(conn, ferr.ErrInvalidConnection)
			return nil, nil, ferr.ErrHealthCheckFailed("health check failed")
		}
	}

	// 应用获取钩子
	if pdb.hooks != nil && pdb.hooks.OnGet != nil {
		if err := pdb.hooks.OnGet(ctx, sqlConn); err != nil {
			pdb.pool.Put(conn, err)
			return nil, nil, err
		}
	}

	return sqlConn, conn, nil
}

// PutConn 将连接归还给池，并应用归还钩子
func (pdb *PooledDB) PutConn(conn pool.Connection, err error) {
	if conn != nil && pdb.pooled {
		// 应用归还钩子
		if pdb.hooks != nil && pdb.hooks.OnPut != nil {
			if sqlConn, ok := conn.Raw().(*sql.DB); ok {
				putErr := pdb.hooks.OnPut(sqlConn, err)
				if putErr != nil {
					err = putErr
				}
			}
		}

		pdb.pool.Put(conn, err)
	}
}

// Close 关闭连接池
func (pdb *PooledDB) Close() error {
    if !pdb.pooled {
        // 应用关闭钩子
        if pdb.hooks != nil && pdb.hooks.OnClose != nil {
            if err := pdb.hooks.OnClose(pdb.sqlDB); err != nil {
                return err
            }
        }
        return pdb.sqlDB.Close()
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // 关闭连接池之前也调用OnClose钩子
    if pdb.hooks != nil && pdb.hooks.OnClose != nil {
        if err := pdb.hooks.OnClose(pdb.sqlDB); err != nil {
            return err
        }
    }
    
    return pdb.pool.Shutdown(ctx)
}

// Stats 返回连接池统计信息
func (pdb *PooledDB) Stats() pool.Stats {
	if !pdb.pooled {
		return pool.Stats{}
	}
	return pdb.pool.Stats()
}

// IsPooled 检查是否启用了连接池
func (pdb *PooledDB) IsPooled() bool {
	return pdb.pooled
}
