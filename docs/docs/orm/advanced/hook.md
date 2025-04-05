# 钩子函数

WebFrame ORM 提供了钩子函数机制，允许用户深度介入数据库连接的生命周期，实现连接监控、定制化健康检查和高级调试等功能。

## 钩子函数概述

钩子函数（Hooks）是一种强大的回调机制，允许您在数据库连接生命周期的关键点执行自定义代码。这对于实现以下需求尤其有用：

- 连接获取和释放的监控和统计
- 自定义连接健康检查
- 连接问题的诊断和日志记录
- 资源分配和清理的跟踪
- 性能监测和优化

WebFrame ORM 的钩子函数与连接池系统深度集成，无论是单一连接还是在高并发场景下都能发挥重要作用。

## 可用的钩子函数

WebFrame ORM 提供了四种类型的钩子函数，覆盖了数据库连接生命周期的关键节点：

```go
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
```

### 1. OnGet 钩子

在从连接池获取连接时调用，可用于：
- 记录连接获取的时间和频率
- 在连接被使用前进行预处理或验证
- 注入追踪或监控标识符
- 如果返回错误，则阻止使用该连接

### 2. OnPut 钩子

在归还连接到连接池时调用，可用于：
- 记录连接使用的持续时间
- 检查和记录查询执行过程中的错误
- 重置连接状态
- 在特定条件下决定是否关闭而非重用连接

### 3. OnCheckHealth 钩子

用于检查连接是否健康，可用于：
- 实现自定义连接活跃度检查
- 添加特定于应用的验证逻辑
- 处理特殊连接状态的验证

### 4. OnClose 钩子

在连接关闭时调用，可用于：
- 执行资源清理操作
- 记录连接关闭事件
- 发送指标和统计信息

## 配置钩子函数

要配置连接钩子，使用 `WithConnHooks` 方法作为数据库选项：

```go
import (
    "context"
    "database/sql"
    "log"
    "time"
    
    "github.com/fyerfyer/fyer-webframe/orm"
)

func main() {
    // 创建自定义钩子
    hooks := &orm.ConnHooks{
        OnGet: func(ctx context.Context, conn *sql.DB) error {
            log.Printf("Connection acquired from pool")
            return nil
        },
        
        OnPut: func(conn *sql.DB, err error) error {
            if err != nil {
                log.Printf("Connection returned with error: %v", err)
            } else {
                log.Printf("Connection returned to pool")
            }
            return nil
        },
        
        OnCheckHealth: func(conn *sql.DB) bool {
            ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
            defer cancel()
            return conn.PingContext(ctx) == nil
        },
        
        OnClose: func(conn *sql.DB) error {
            log.Printf("Connection closed")
            return nil
        },
    }
    
    // 创建数据库连接，并应用钩子
    db, err := orm.OpenDB(
        "mysql", 
        "user:password@tcp(localhost:3306)/dbname",
        "mysql",
        orm.WithConnectionPool(
            orm.WithPoolMaxIdle(10),
            orm.WithPoolMaxActive(100),
        ),
        orm.WithConnHooks(hooks),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 使用数据库...
}
```

## 实际使用场景

### 连接性能监控

使用钩子函数测量和记录连接的获取和使用时间：

```go
hooks := &orm.ConnHooks{
    OnGet: func(ctx context.Context, conn *sql.DB) error {
        // 在上下文中存储获取时间
        ctx = context.WithValue(ctx, "conn_start_time", time.Now())
        return nil
    },
    
    OnPut: func(conn *sql.DB, err error) error {
        if startTime, ok := ctx.Value("conn_start_time").(time.Time); ok {
            duration := time.Since(startTime)
            if duration > time.Second {
                log.Printf("SLOW CONNECTION USAGE: %v", duration)
            }
            
            // 更新指标
            metrics.RecordConnectionDuration(duration)
        }
        return nil
    },
}
```

### 高级健康检查

实现更复杂的健康检查逻辑：

```go
hooks := &orm.ConnHooks{
    OnCheckHealth: func(conn *sql.DB) bool {
        ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
        defer cancel()
        
        // 首先进行基本的 Ping 检查
        if err := conn.PingContext(ctx); err != nil {
            return false
        }
        
        // 然后执行简单查询以验证更多功能
        var result int
        row := conn.QueryRowContext(ctx, "SELECT 1")
        if err := row.Scan(&result); err != nil {
            log.Printf("Health check query failed: %v", err)
            return false
        }
        
        // 检查复制延迟（适用于主从数据库设置）
        if isReplicaDB(conn) {
            replicationLag, err := checkReplicationLag(ctx, conn)
            if err != nil || replicationLag > 30 {
                log.Printf("Replica lag too high: %d seconds", replicationLag)
                return false
            }
        }
        
        return result == 1
    },
}
```

### 连接问题诊断

使用钩子函数帮助诊断连接问题：

```go
hooks := &orm.ConnHooks{
    OnGet: func(ctx context.Context, conn *sql.DB) error {
        activeConnections.Inc() // 增加活动连接计数
        
        // 记录连接池状态
        stats := db.PoolStats()
        log.Printf("Connection acquired. Active: %d, Idle: %d, WaitCount: %d", 
                  stats.Active, stats.Idle, stats.WaitCount)
        
        return nil
    },
    
    OnPut: func(conn *sql.DB, err error) error {
        activeConnections.Dec() // 减少活动连接计数
        
        if err != nil {
            // 记录连接错误
            connectionErrors.Inc()
            log.Printf("Connection returned with error: %v", err)
            
            // 如果是致命错误，返回错误，连接池会关闭这个连接
            if isFatalError(err) {
                return err
            }
        }
        
        return nil
    },
}
```

### 事务跟踪

跟踪事务的开始和结束：

```go
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithConnHooks(&orm.ConnHooks{
        OnGet: func(ctx context.Context, conn *sql.DB) error {
            if txID := ctx.Value("transaction_id"); txID != nil {
                log.Printf("Transaction %v acquired connection", txID)
            }
            return nil
        },
        OnPut: func(conn *sql.DB, err error) error {
            if txID := ctx.Value("transaction_id"); txID != nil {
                if err != nil {
                    log.Printf("Transaction %v returned connection with error: %v", txID, err)
                } else {
                    log.Printf("Transaction %v returned connection", txID)
                }
            }
            return nil
        },
    }),
)

// 使用事务
ctx = context.WithValue(context.Background(), "transaction_id", uuid.New().String())
err = db.Tx(ctx, func(tx *orm.Tx) error {
    // 执行事务操作...
    return nil
})
```

## 连接跟踪

WebFrame ORM 还提供了 `ConnectionTracker` 用于跟踪查询结果集与其底层连接之间的关系：

```go
// 创建连接跟踪器
tracker := orm.NewConnectionTracker()

// 跟踪连接和结果集
tracker.TrackRows(rows, conn)

// 在结果集使用完成后释放连接
defer tracker.ReleaseRows(rows, pooledDB)
```

这在处理大型结果集时特别有用，确保连接在结果集使用完成后才归还给连接池。
