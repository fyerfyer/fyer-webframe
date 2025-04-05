# Sharding

WebFrame ORM 提供了对数据分片的支持，以构建高性能、高可扩展的数据库访问层。

## 分片基础概念

### 什么是数据分片？

数据分片是一种水平分区技术，它将单个数据库表的行分布到多个物理数据库或表中。通过数据分片，您可以：

- 提高查询性能
- 增强系统可扩展性
- 更好地管理大数据量
- 减少单个节点的负载

### 关键概念

在 WebFrame ORM 的分片系统中，有几个核心概念：

- **分片键（Shard Key）**：用于确定数据应路由到哪个分片的字段
- **分片策略（Sharding Strategy）**：决定如何基于分片键分配数据的算法
- **分片路由（Shard Router）**：负责计算具体记录应存储在哪个分片的组件
- **分片表（Sharded Table）**：跨多个数据库实例的逻辑上相同的表


## 总体架构

分片系统的整体架构遵循分层设计原则：

1. **内部实现层** (`internal/sharding`): 提供核心的分片算法和路由逻辑
2. **对外 API 层** (orm 包): 提供与 ORM 集成的用户友好接口
3. **中间件层**: 通过中间件自动拦截和转发查询到正确的分片

```
┌─────────────────────────────────────────────────────┐
│                  用户应用代码                        │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│         ShardingClient / Collection API             │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│               ShardingMiddleware                    │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│                   ShardingDB                        │
└──┬─────────────────────────────────────────────┬────┘
   │                                             │
┌──▼─────────────────────┐   ┌──────────────────▼────┐
│    ShardingManager     │   │     DefaultRouter     │
└──┬─────────────────────┘   └──────────────────┬────┘
   │                                            │
┌──▼─────────────────────────────────────────────────┐
│                 分片策略 (Strategy)                │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌─────────┐  │
│  │HashStrategy│RangeStrategy│DateStrategy│ModStrategy│
│  └──────────┘ └──────────┘ └────────┘ └─────────┘  │
└──────────────────────────────────────────────────┬─┘
                                                   │
┌──────────────────────────────────────────────────▼─┐
│                  物理分片数据库                    │
└────────────────────────────────────────────────────┘
```

## 关键设计模式

WebFrame ORM 分片系统采用多种设计模式来实现灵活性和可扩展性：

### 1. 策略模式

分片策略使用策略模式，通过 `Strategy` 接口统一定义路由行为，并提供多种具体实现：

```go
type Strategy interface {
    Route(key interface{}) (dbIndex, tableIndex int, err error)
    GetShardName(dbIndex, tableIndex int) (dbName, tableName string, err error)
}
```

四种策略实现（哈希、范围、日期、取模）都继承自 `BaseStrategy` 基类，并提供各自的路由算法。

### 2. 适配器模式

通过 `shardingStrategyAdapter` 将内部的分片策略适配为对外的 `ShardingStrategy` 接口，实现内外层的解耦。

### 3. 代理模式

`ShardedDB` 作为普通 `DB` 的代理，拦截查询请求并将其路由到正确的分片，对调用方透明。

### 4. 建造者模式

所有分片策略和路由器都使用建造者模式，通过流畅的 API 进行配置：

```go
hashStrategy := WithHashStrategy("user_db_", 4, "user_", 8, "UserID")

router := NewShardingRouter().
    WithCacheSize(1000).
    WithCacheExpiration(time.Minute)
```

## 对象依赖关系

分片系统的主要对象及其依赖关系如下：

```
ShardingDB
  ├── 原始DB (默认数据库)
  ├── ShardingManager
  │     ├── 分片DB映射 (shards map[string]*DB)
  │     └── Router
  └── ShardingMiddleware
        └── 提取分片键并路由查询

Router
  ├── 模型信息映射 (models map[string]*ModelInfo)
  ├── 路由缓存 (routeCache sync.Map)
  └── Strategy策略映射

Strategy
  ├── BaseStrategy (共享基础实现)
  │     ├── HashStrategy (基于哈希分片)
  │     ├── RangeStrategy (基于范围分片)
  │     ├── DateStrategy (基于日期分片)
  │     └── ModStrategy (基于取模分片)
  └── 计算分片位置
```

## 工作流程

一个典型的分片查询处理流程如下：

1. **查询拦截**：`ShardingMiddleware` 拦截数据库查询
2. **分片键提取**：从查询条件中提取分片键值
3. **路由计算**：使用注册的分片策略计算目标分片位置
4. **缓存检查**：检查路由缓存，避免重复计算
5. **SQL 重写**：根据分片结果重写查询的表名
6. **分片选择**：选择正确的分片数据库连接
7. **执行查询**：在目标分片上执行修改后的查询
8. **结果返回**：将查询结果返回给调用方

## 分片策略

WebFrame ORM 提供了四种内置分片策略，每种策略适用于不同的场景：

### 1. 哈希分片策略（Hash Strategy）

基于分片键的哈希值将数据均匀分布到各个分片。

**适用场景**：需要均匀分布数据，且无需按范围查询时。

```go
// 创建基于哈希的分片策略
hashStrategy := WithHashStrategy("user_db_", 4, "user_", 8, "UserID")

// 参数说明:
// - "user_db_" - 数据库名称前缀
// - 4 - 数据库数量
// - "user_" - 表名前缀
// - 8 - 每个数据库的表数量
// - "UserID" - 分片键
```

### 2. 范围分片策略（Range Strategy）

基于分片键的数值范围将数据分配到各个分片。

**适用场景**：需要基于连续范围值进行高效查询时，例如按日期范围查询。

```go
// 创建基于范围的分片策略
ranges := []int64{1000, 2000, 3000, 4000} // 范围边界值
rangeStrategy := WithRangeStrategy("order_db_", 4, "order_", 5, "OrderID", ranges)

// 例如，ID在[0,1000)路由到分片0，[1000,2000)路由到分片1，依此类推
```

### 3. 日期分片策略（Date Strategy）

基于日期字段将数据分配到各个分片，支持按日、周、月、年分片。

**适用场景**：时间序列数据，如日志、交易记录等。

```go
// 创建基于日期的分片策略
dateStrategy := WithDateStrategy("log_db_", 4, "log_", 12, "CreateTime", "monthly").
    WithStartTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

// 日期格式支持: "daily", "weekly", "monthly", "yearly"
```

### 4. 取模分片策略（Mod Strategy）

使用分片键对分片数量取模，是一种简单但有效的分片方法。

**适用场景**：需要简单、确定性分片规则，且数据均匀分布时。

```go
// 创建基于取模的分片策略
modStrategy := WithModStrategy("product_db_", 3, "product_", 5, "ProductID")

// 例如，ID = 1001: 
// 数据库索引 = 1001 % 3 = 1 (数据库名为 product_db_1)
// 表索引 = (1001 / 3) % 5 = 3 (表名为 product_3)
```

## 配置分片

### 创建分片数据库

使用 `NewShardingDB` 创建一个支持分片的数据库实例：

```go
// 1. 创建基础数据库连接
db, err := orm.OpenDB("mysql", "root:password@tcp(localhost:3306)/default_db", "mysql")
if err != nil {
    log.Fatal(err)
}

// 2. 创建路由器
router := orm.NewShardingRouter()

// 3. 创建分片数据库
shardDB := orm.NewShardingDB(db, router)

// 4. 注册分片策略
shardDB.RegisterShardStrategy("User", 
    orm.WithHashStrategy("user_db_", 4, "user_", 8, "UserID"), 
    "user_db_0") // 默认分片数据库

// 5. 配置分片连接
err = shardDB.ConfigureShards(map[string]orm.ShardConfig{
    "user_db_0": {
        Driver: "mysql",
        DSN: "root:password@tcp(localhost:3306)/user_db_0",
        MaxIdleConn: 10,
        MaxOpenConn: 100,
    },
    "user_db_1": {
        Driver: "mysql",
        DSN: "root:password@tcp(localhost:3306)/user_db_1",
        MaxIdleConn: 10,
        MaxOpenConn: 100,
    },
    // 更多分片配置...
})
if err != nil {
    log.Fatal(err)
}

// 6. 启用分片
shardDB.EnableSharding()
```

### 通过客户端 API 使用分片

使用客户端 API 可以更简洁地操作分片：

```go
// 创建分片客户端
client := db.NewClient()
shardClient := client.AsShardingClient()

// 注册分片策略
shardClient.RegisterShardStrategy("Order", 
    orm.WithModStrategy("order_db_", 3, "order_", 5, "OrderID"),
    "order_db_0")

// 使用分片集合
orderCollection := shardClient.ShardedCollection(&Order{})

// 基于分片键查询
order, err := orderCollection.Find(ctx, orm.Col("OrderID").Eq(1001))
```

## 使用分片进行数据操作

### 查询操作

在分片环境中执行查询，ORM 会自动路由到正确的分片：

```go
// 1. 使用选择器 API
user, err := orm.RegisterSelector[User](shardDB).
    Select().
    Where(orm.Col("UserID").Eq(1001)).  // 包含分片键的条件
    Get(ctx)

// 2. 使用客户端 API 
userCollection := shardClient.Collection(&User{})
user, err := userCollection.Find(ctx, orm.Col("UserID").Eq(1001))
```

### 插入操作

插入时必须包含分片键值以确定数据路由：

```go
// 1. 使用插入器 API
newUser := &User{UserID: 1002, Name: "Alice", Age: 30}
result, err := orm.RegisterInserter[User](shardDB).
    Insert(nil, newUser).
    Exec(ctx)

// 2. 使用客户端 API
userCollection := shardClient.Collection(&User{})
result, err := userCollection.Insert(ctx, newUser)
```

### 更新操作

更新操作也需要分片键来确定路由：

```go
// 1. 使用更新器 API
result, err := orm.RegisterUpdater[User](shardDB).
    Update().
    Set(orm.Col("Age"), 31).
    Where(orm.Col("UserID").Eq(1002)).  // 必须包含分片键
    Exec(ctx)

// 2. 使用客户端 API
userCollection := shardClient.Collection(&User{})
result, err := userCollection.Update(ctx, 
    map[string]interface{}{"Age": 31},
    orm.Col("UserID").Eq(1002))
```

### 删除操作

删除同样需要分片键来确定路由：

```go
// 1. 使用删除器 API
result, err := orm.RegisterDeleter[User](shardDB).
    Delete().
    Where(orm.Col("UserID").Eq(1002)).  // 必须包含分片键
    Exec(ctx)

// 2. 使用客户端 API
userCollection := shardClient.Collection(&User{})
result, err := userCollection.Delete(ctx, orm.Col("UserID").Eq(1002))
```

## 跨分片操作

有时候需要在多个分片上执行操作，WebFrame ORM 提供了便捷的工具：

### 在所有分片上执行相同操作

```go
// 在所有用户分片上执行分析查询
errors := shardDB.ExecuteOnAllShards(ctx, func(db *orm.DB) error {
    result, err := db.Exec(ctx, "ANALYZE TABLE user")
    if err != nil {
        return err
    }
    log.Printf("Analyzed table on shard: %v", result)
    return nil
})

// 检查是否有错误发生
for i, err := range errors {
    if err != nil {
        log.Printf("Error on shard %d: %v", i, err)
    }
}
```

### 在特定分片上执行操作

```go
// 在特定分片上执行操作
err := shardClient.ExecuteOnShard(ctx, "user_db_1", func(db *orm.DB) error {
    rows, err := db.Raw(ctx, "SELECT COUNT(*) FROM user_1")
    if err != nil {
        return err
    }
    defer rows.Close()
    
    var count int
    if rows.Next() {
        if err := rows.Scan(&count); err != nil {
            return err
        }
    }
    log.Printf("User count on shard user_db_1: %d", count)
    return nil
})
```

## 分片路由原理

WebFrame ORM 分片系统的核心是其路由机制，它确定数据应该路由到哪个分片：

1. **路由计算**：根据分片键值和分片策略计算分片位置
2. **路由缓存**：高频路由结果会被缓存以提高性能
3. **分片替换**：查询执行前自动替换表名和数据库连接

### 路由过程示例

以取模分片为例，路由过程如下：

1. 分片键值: `OrderID = 1001`
2. 使用取模策略：
    - 数据库索引 = `1001 % 3 = 2` → 选择 `order_db_2` 分片
    - 表索引 = `(1001 / 3) % 5 = 3` → 选择 `order_3` 表
3. 替换原始 SQL 中的表名：`FROM order` → `FROM order_3`
4. 使用 `order_db_2` 分片连接执行查询

## 分片中的事务处理

分片环境中的事务具有特殊性，因为它们通常需要跨多个数据库实例：

### 单分片事务

如果事务只涉及单个分片，WebFrame ORM 会自动处理：

```go
// 在单个分片上的事务
err := shardDB.Tx(ctx, func(tx *orm.Tx) error {
    // 这些操作在同一个分片上，支持标准事务
    result1, err := orm.RegisterUpdater[Order](tx).
        Update().
        Set(orm.Col("Status"), "processing").
        Where(orm.Col("OrderID").Eq(1001)).
        Exec(ctx)
    if err != nil {
        return err  // 自动回滚
    }
    
    result2, err := orm.RegisterInserter[OrderLog](tx).
        Insert(nil, &OrderLog{
            OrderID: 1001,
            Action: "status_change",
            Time: time.Now(),
        }).
        Exec(ctx)
    if err != nil {
        return err  // 自动回滚
    }
    
    return nil  // 提交事务
}, nil)
```

### 跨分片事务注意事项

WebFrame ORM 不直接支持跨分片的分布式事务，因为这需要更复杂的协调机制。对于跨分片场景，您需要实现应用层逻辑来处理数据一致性：

1. **分阶段提交模式**：使用准备-提交-确认阶段
2. **最终一致性模式**：使用消息队列或事件系统确保数据最终一致
3. **补偿事务模式**：当部分操作失败时执行补偿操作

## 分片统计和监控

WebFrame ORM 分片系统提供了统计信息，帮助您监控分片性能：

```go
// 获取分片统计信息
stats := shardDB.GetShardingManager().GetStats()

// 查看各分片使用情况
for shard, count := range stats.RouteCount {
    log.Printf("Shard %s: %d requests", shard, count)
}

// 查看缓存命中率
cacheHitRate := float64(stats.CacheHit) / float64(stats.CacheHit + stats.CacheMiss)
log.Printf("Route cache hit rate: %.2f%%", cacheHitRate * 100)

// 查看最后访问时间
for shard, lastAccess := range stats.LastAccessTime {
    log.Printf("Shard %s last access: %v", shard, lastAccess)
}
```