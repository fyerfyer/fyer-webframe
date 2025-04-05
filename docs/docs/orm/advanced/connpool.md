# Conn Pool

WebFrame ORM 内置了数据库连接池管理功能，可以显著提高应用程序性能、增强可靠性并降低资源消耗。

## 连接池概述

数据库连接池是一种重要的性能优化技术，特别是在高并发应用中。连接池的主要优势包括：

- **减少连接开销**：创建数据库连接是一个相对昂贵的操作，连接池通过复用已有连接来减少这一开销
- **提高并发处理能力**：预先创建连接可以处理突发的请求增长
- **管理资源使用**：限制同时打开的数据库连接数，防止资源耗尽
- **提供健康检查**：检测并替换失效的连接

WebFrame ORM 使用基于 `fyerfyer/fyer-kit/pool` 包实现的高性能连接池，支持多种配置选项和监控功能。

## 启用连接池

### 基本用法

在创建 ORM 实例时启用连接池：

```go
// 使用连接池选项初始化 ORM
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithConnectionPool(
        orm.WithPoolMaxIdle(10),    // 最大空闲连接数
        orm.WithPoolMaxActive(100), // 最大活动连接数
    ),
)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### 快捷方式

WebFrame ORM 提供了一个便捷的短语法来设置连接池的基本参数：

```go
// 使用快捷方法配置连接池
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithPoolSize(10, 100), // 参数: 最大空闲连接数和最大活动连接数
)
```

## 详细配置选项

WebFrame ORM 提供多种连接池配置选项，可根据应用需求进行精细调整：

```go
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithConnectionPool(
        orm.WithPoolMaxIdle(10),                  // 最大空闲连接数
        orm.WithPoolMaxActive(100),               // 最大活动连接数(0表示无限制)
        orm.WithPoolMaxIdleTime(5 * time.Minute), // 连接最大空闲时间
        orm.WithPoolMaxLifetime(30 * time.Minute), // 连接最大生命周期
        orm.WithPoolInitialSize(5),               // 初始连接数
        orm.WithPoolWaitTimeout(3 * time.Second), // 等待可用连接的超时时间
        orm.WithPoolDialTimeout(2 * time.Second), // 连接超时时间
        orm.WithPoolHealthCheck(customHealthCheck), // 自定义健康检查函数
    ),
)
```

### 配置选项详解

| 选项 | 描述 | 默认值 | 推荐值 |
|------|------|--------|--------|
| `MaxIdle` | 最大空闲连接数 | 10 | 与常规并发请求数相匹配 |
| `MaxActive` | 最大活动连接数 | 100 | 根据预期峰值负载设置 |
| `MaxIdleTime` | 连接最大空闲时间 | 5分钟 | 根据数据库超时设置 |
| `MaxLifetime` | 连接最大生命周期 | 30分钟 | 避免资源泄漏 |
| `InitialSize` | 初始连接数 | 5 | 根据启动时预期负载 |
| `WaitTimeout` | 等待连接超时 | 3秒 | 根据应用的超时容忍度 |
| `DialTimeout` | 建立连接超时 | 2秒 | 根据网络条件 |

## 连接健康检查

连接池包含内置的健康检查机制，确保返回给应用程序的连接是有效的：

```go
// 默认的健康检查函数
func defaultHealthCheck(db *sql.DB) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    return db.PingContext(ctx) == nil
}

// 自定义健康检查函数
func customHealthCheck(db *sql.DB) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    
    // 执行简单查询检查连接
    var result int
    err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
    return err == nil && result == 1
}

// 配置自定义健康检查
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithConnectionPool(
        orm.WithPoolHealthCheck(customHealthCheck),
    ),
)
```

## 监控连接池

WebFrame ORM 提供了监控连接池状态的功能，便于了解池的使用情况和及时调整配置：

```go
// 获取连接池统计信息
stats := db.PoolStats()

// 打印连接池使用情况
log.Printf("Pool stats: Active=%d, Idle=%d, WaitCount=%d, WaitTime=%v, IdleTimeout=%v",
    stats.Active, stats.Idle, stats.WaitCount, stats.WaitDuration, stats.IdleTimeout)
```

可以监控的指标包括：
- `Active`：当前活跃的连接数
- `Idle`：当前空闲的连接数
- `WaitCount`：等待连接的总请求数
- `WaitDuration`：等待连接的总时间
- `IdleTimeout`：因空闲超时关闭的连接数
- `LifetimeTimeout`：因生命周期超时关闭的连接数

## 事务与连接池

WebFrame ORM 会自动处理事务中的连接管理，确保事务使用同一个连接，并在事务完成后正确地将连接归还给池：

```go
// 启用连接池的情况下使用事务
err = db.Tx(ctx, func(tx *orm.Tx) error {
    // 所有在事务中的操作都会使用同一个连接
    result1, err := orm.RegisterUpdater[Account](tx).
        Update().
        Set(orm.Col("Balance"), orm.Raw("Balance - ?", 100)).
        Where(orm.Col("UserID").Eq(1)).
        Exec(ctx)
    if err != nil {
        return err  // 自动回滚并归还连接
    }
    
    result2, err := orm.RegisterUpdater[Account](tx).
        Update().
        Set(orm.Col("Balance"), orm.Raw("Balance + ?", 100)).
        Where(orm.Col("UserID").Eq(2)).
        Exec(ctx)
    if err != nil {
        return err  // 自动回滚并归还连接
    }
    
    return nil  // 提交事务，归还连接
}, nil)
```

当事务提交或回滚时，连接会自动归还给连接池。

## 使用现有连接池

如果您已经有了一个 `fyerfyer/fyer-kit/pool.Pool` 实例，可以直接使用它：

```go
import (
    "github.com/fyerfyer/fyer-kit/pool"
    "github.com/fyerfyer/fyer-webframe/orm"
)

// 使用已存在的连接池
func useExistingPool(existingPool pool.Pool) (*orm.DB, error) {
    db, err := orm.Open(sqlDB, "mysql", orm.WithExistingPool(existingPool))
    if err != nil {
        return nil, err
    }
    return db, nil
}
```

## 多数据库连接池

在复杂应用中，可能需要管理多个数据库连接池：

```go
// 主数据库连接池
mainDB, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/maindb",
    "mysql",
    orm.WithPoolSize(10, 50),
)
if err != nil {
    log.Fatal(err)
}
defer mainDB.Close()

// 读取副本连接池
replicaDB, err := orm.OpenDB(
    "mysql", 
    "readonly:password@tcp(replica.example.com:3306)/maindb",
    "mysql",
    orm.WithPoolSize(20, 100), // 为读操作配置更大的池
)
if err != nil {
    log.Fatal(err)
}
defer replicaDB.Close()

// 使用特定的池
func getUserFromReplica(ctx context.Context, userID int64) (*User, error) {
    return orm.RegisterSelector[User](replicaDB).
        Select().
        Where(orm.Col("ID").Eq(userID)).
        Get(ctx)
}

func updateUserInMain(ctx context.Context, user *User) error {
    _, err := orm.RegisterUpdater[User](mainDB).
        Update().
        SetMulti(map[string]interface{}{
            "Name": user.Name,
            "Email": user.Email,
        }).
        Where(orm.Col("ID").Eq(user.ID)).
        Exec(ctx)
    return err
}
```

## 示例：完整的连接池配置

以下是一个生产环境中连接池配置的综合示例：

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/fyerfyer/fyer-webframe/orm"
)

func main() {
    // 创建并配置数据库连接池
    db, err := orm.OpenDB(
        "mysql",
        "user:password@tcp(localhost:3306)/production_db?parseTime=true",
        "mysql",
        orm.WithConnectionPool(
            orm.WithPoolMaxIdle(10),                  // 保持适量的空闲连接
            orm.WithPoolMaxActive(100),               // 限制最大活动连接
            orm.WithPoolMaxIdleTime(5 * time.Minute), // 避免空闲连接长期占用资源
            orm.WithPoolMaxLifetime(30 * time.Minute), // 避免连接使用过长
            orm.WithPoolInitialSize(5),               // 应用启动时创建一些连接
            orm.WithPoolWaitTimeout(5 * time.Second), // 设置等待超时
            orm.WithPoolDialTimeout(3 * time.Second), // 设置连接超时
        ),
    )
    if err != nil {
        log.Fatalf("Failed to create database: %v", err)
    }
    defer db.Close()

    // 创建一个定期监控连接池的协程
    go monitorConnectionPool(db)

    // 应用主逻辑...
}

// 监控连接池状态
func monitorConnectionPool(db *orm.DB) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        stats := db.PoolStats()
        log.Printf("DB Pool Stats: Active=%d, Idle=%d, WaitCount=%d, WaitTime=%v",
            stats.Active, stats.Idle, stats.WaitCount, stats.WaitDuration)
        
        // 如果等待连接的次数很多，可能需要增加连接池大小
        if stats.WaitCount > 100 {
            log.Printf("Warning: High wait count (%d) for database connections", 
                stats.WaitCount)
        }
    }
}
```