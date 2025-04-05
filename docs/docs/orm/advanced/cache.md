# Cache Layer

WebFrame ORM 提供了强大的查询缓存系统，让您能够大幅提升应用性能，尤其是对于频繁执行的相同查询。本文详细介绍 ORM 缓存机制的工作原理、配置方法和最佳实践。

## 缓存系统概述

WebFrame ORM 缓存系统的核心组件：

- **Cache 接口**：定义缓存操作的基本方法
- **CacheManager**：管理缓存策略和配置
- **ModelCacheConfig**：针对特定模型的缓存配置
- **缓存中间件**：自动拦截和缓存查询结果

## 架构设计

缓存系统的架构采用分层设计，包含以下核心组件：

```
┌─────────────────────────────────────┐
│            ORM 客户端 API           │
└───────────────────┬─────────────────┘
                    │
┌───────────────────▼─────────────────┐
│           查询构建与执行层           │
└───────────────────┬─────────────────┘
                    │
┌───────────────────▼─────────────────┐
│          缓存中间件 (拦截查询)       │
└───────────────────┬─────────────────┘
                    │
┌───────────────────▼─────────────────┐
│            缓存管理器                │
│  ┌────────────┐   ┌────────────┐    │
│  │ 模型配置   │   │ 键生成器   │    │
│  └────────────┘   └────────────┘    │
└───────────────────┬─────────────────┘
                    │
┌───────────────────▼─────────────────┐
│          缓存提供者接口              │
└───────────────────┬─────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
┌───────▼───────┐       ┌───────▼───────┐
│  内存缓存     │       │   自定义缓存   │
└───────────────┘       └───────────────┘
```

## 设计模式

缓存系统使用了多种设计模式来实现其功能：

### 1. 策略模式

缓存系统通过 `Cache` 接口定义了缓存操作的标准行为，允许不同的缓存实现。

```go
type Cache interface {
    Get(ctx context.Context, key string, value interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeleteByTags(ctx context.Context, tags ...string) error
    Clear(ctx context.Context) error
}
```

用户可以选择内存缓存、Redis 缓存或自定义缓存实现，而不会影响上层代码。

### 2. 中间件模式

缓存功能通过中间件集成到查询流程中，遵循责任链模式：

```go
func CacheMiddleware(cacheManager *CacheManager) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
            // 1. 检查是否应该缓存
            // 2. 尝试从缓存获取
            // 3. 缓存未命中时调用下一个处理器
            // 4. 缓存查询结果
        })
    }
}
```

### 3. 构建器模式

缓存配置使用构建器模式，允许流式配置：

```go
cacheManager := NewCacheManager(memCache).
    WithDefaultTTL(5 * time.Minute).
    WithKeyPrefix("myapp:")
```

### 4. 装饰器模式

`WithCache()` 和 `WithoutCache()` 方法用于装饰选择器对象，动态添加或禁用缓存功能：

```go
// 启用缓存的选择器
user, err := orm.RegisterSelector[User](db).
    Select().
    Where(orm.Col("ID").Eq(123)).
    WithCache().
    WithCacheTags("user:123").
    Get(ctx)
```

## 组件依赖关系

缓存系统的组件之间存在清晰的依赖关系：

1. **CacheManager** 依赖于 **Cache** 接口和 **KeyGenerator**
2. **CacheMiddleware** 依赖于 **CacheManager**
3. **MemoryCache** 实现了 **Cache** 接口
4. **DB** 包含 **CacheManager** 作为成员
5. **Client** 访问 **DB** 的缓存功能

## 缓存键生成与管理

缓存键生成是缓存系统的核心部分：

```
┌────────────────────────────────────────┐
│              KeyGenerator              │
├────────────────────────────────────────┤
│ Generate(model, operation, query, args)│
│ GenerateWithTags(model, op, query, tags)│
│ BuildTagKey(tag)                       │
└────────────────────────────────────────┘
```

主要实现位于 key.go 中，提供了：

1. **前缀管理**：为不同应用或环境添加前缀
2. **一致性算法**：确保相同查询生成相同的键
3. **键长度限制**：防止超长键导致的性能问题
4. **参数编码**：将查询参数编码到键中，确保唯一性

## 缓存接口

WebFrame ORM 定义了一个通用缓存接口，允许集成任何支持该接口的缓存实现：

```go
// Cache 定义缓存接口，用户可以实现此接口来提供自定义缓存
type Cache interface {
    // Get 从缓存获取值，如果不存在返回 ErrCacheMiss
    Get(ctx context.Context, key string, value interface{}) error

    // Set 设置缓存值
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

    // Delete 删除缓存值
    Delete(ctx context.Context, key string) error

    // DeleteByTags 通过标签批量删除缓存
    DeleteByTags(ctx context.Context, tags ...string) error

    // Clear 清空缓存
    Clear(ctx context.Context) error
}
```

## 内存缓存实现

WebFrame ORM 内置了一个简单但高效的内存缓存实现 `MemoryCache`：

```go
// 创建内存缓存
memCache := NewMemoryCache(
    WithGCInterval(5 * time.Minute),  // 垃圾回收间隔
    WithMaxEntries(10000),            // 最大缓存条目数
)

// 配置DB使用内存缓存
db, err := orm.OpenDB(
    "mysql",
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithDBMiddlewareCache(memCache), // 设置缓存
)
```

内存缓存的特点：

- 自动过期：支持基于 TTL 的缓存过期
- 定期清理：后台 goroutine 清理过期项
- 容量限制：可配置最大缓存条目数，避免内存溢出
- 标签索引：支持通过标签快速定位和失效相关缓存

## 缓存键生成与管理

缓存键生成是缓存系统的核心部分：

```
┌────────────────────────────────────────┐
│              KeyGenerator              │
├────────────────────────────────────────┤
│ Generate(model, operation, query, args)│
│ GenerateWithTags(model, op, query, tags)│
│ BuildTagKey(tag)                       │
└────────────────────────────────────────┘
```

主要实现位于 key.go 中，提供了：

1. **前缀管理**：为不同应用或环境添加前缀
2. **一致性算法**：确保相同查询生成相同的键
3. **键长度限制**：防止超长键导致的性能问题
4. **参数编码**：将查询参数编码到键中，确保唯一性

## 缓存配置和管理

### 缓存管理器

`CacheManager` 是管理缓存策略和配置的核心组件：

```go
// 创建DB并配置缓存
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname", 
    "mysql",
    orm.WithDBCache(NewMemoryCache()),          // 设置缓存实现
    orm.WithDefaultCacheTTL(5 * time.Minute),   // 设置默认TTL
    orm.WithCacheKeyPrefix("myapp:"),           // 设置缓存键前缀
)
```

### 模型缓存配置

为特定模型配置缓存策略：

```go
// 为User模型配置缓存
db.SetModelCacheConfig("User", &ModelCacheConfig{
    Enabled: true,                     // 启用缓存
    TTL:     10 * time.Minute,         // 缓存10分钟
    Tags:    []string{"user", "auth"}, // 关联标签
})
```

`ModelCacheConfig` 可定义的选项：

- **Enabled**：是否启用缓存
- **TTL**：缓存过期时间
- **Tags**：缓存关联的标签
- **KeyGenerator**：自定义缓存键生成函数
- **Conditions**：缓存条件，决定哪些查询应被缓存

使用客户端 API 配置缓存：

```go
client := db.NewClient()

// 为User模型配置缓存
client.SetModelCacheConfig("User", &ModelCacheConfig{
    Enabled: true,
    TTL:     10 * time.Minute,
})
```

## 在查询中使用缓存

### 选择器缓存

选择器（Selector）提供了细粒度的缓存控制：

```go
// 启用缓存并设置TTL
user, err := orm.RegisterSelector[User](db).
    Select().
    Where(orm.Col("ID").Eq(123)).
    WithCache().                              // 启用缓存
    WithSelectorCacheTTL(time.Minute).        // 设置TTL
    WithCacheTags("user:123", "profile").     // 设置标签
    Get(ctx)

// 显式禁用缓存
users, err := orm.RegisterSelector[User](db).
    Select().
    Where(orm.Col("Age").Gt(18)).
    WithoutSelectorCache().                   // 禁用缓存
    GetMulti(ctx)
```

### 客户端 API 缓存控制

客户端 API 同样支持缓存控制：

```go
// 创建启用缓存的客户端
cachedClient := client.WithCache()

// 查找单个用户（使用缓存）
user, err := cachedClient.Collection(&User{}).Find(ctx, 
    orm.Col("ID").Eq(123))

// 创建禁用缓存的客户端
noCacheClient := client.WithoutCache()

// 查找所有活跃用户（不使用缓存）
activeUsers, err := noCacheClient.Collection(&User{}).FindAll(ctx,
    orm.Col("Status").Eq("active"))
```

## 缓存标签系统

WebFrame ORM 的缓存实现了标签系统，建立了缓存项之间的逻辑关系：

```go
// 内部数据结构
type MemoryCache struct {
    data       map[string]item
    tagToKeys  map[string]map[string]struct{} // 标签 -> 键集合
    keyToTags  map[string][]string            // 键 -> 标签列表
    // ...其他字段
}
```

这种双向映射使得基于标签的缓存失效操作高效且精确，特别适用于关联数据的更新场景。

标签系统是 WebFrame ORM 缓存的关键特性，它为缓存项建立了逻辑关系，便于批量管理缓存：

```go
// 创建带标签的缓存项
err := memCache.SetWithTags(ctx, "user:profile:123", userData, 
    5*time.Minute, "user:123", "profile")

// 通过标签批量删除缓存，将删除所有关联此标签的缓存项
err := memCache.DeleteByTags(ctx, "user:123")
```

### 有效的标签使用模式

- **模型类型标签**：如 `"user"`, `"product"`
- **实体标签**：如 `"user:123"`, `"product:456"`
- **关系标签**：如 `"user:123:orders"`, `"category:5:products"`
- **操作标签**：如 `"recently_viewed"`, `"popular_products"`

## 缓存粒度控制

缓存系统提供了多级粒度控制：

1. **全局级别**：通过 `CacheManager` 的 `Enable()` 和 `Disable()` 方法
2. **模型级别**：使用 `ModelCacheConfig` 为不同模型定制策略
3. **查询级别**：通过 `WithCache()` 和 `WithoutCache()` 方法
4. **条件级别**：使用 `CacheCondition` 函数判断特定查询是否应被缓存

```go
// 为用户模型配置特定缓存策略
db.SetModelCacheConfig("User", &ModelCacheConfig{
    Enabled: true,
    TTL:     10 * time.Minute,
    Tags:    []string{"user", "auth"},
    Conditions: []CacheCondition{
        // 只缓存按 ID 查询的结果
        func(ctx context.Context, qc *QueryContext) bool {
            return strings.Contains(qc.Query.SQL, "WHERE `id` =")
        },
    },
})
```

## 缓存失效策略

### 自动失效

WebFrame ORM 在执行写操作时自动失效相关缓存：

```go
// 执行更新操作时自动失效相关缓存
result, err := orm.RegisterUpdater[User](db).
    Update().
    Set(orm.Col("Name"), "NewName").
    Where(orm.Col("ID").Eq(123)).
    WithInvalidateCache().                   // 启用缓存失效
    WithInvalidateTags("user:123", "profile"). // 指定失效标签
    Exec(ctx)

// 执行删除操作时自动失效相关缓存
result, err := orm.RegisterDeleter[User](db).
    Delete().
    Where(orm.Col("ID").Eq(123)).
    WithInvalidateCache().                   // 启用缓存失效
    WithInvalidateTags("user:123").          // 指定失效标签
    Exec(ctx)

// 执行插入操作时自动失效相关缓存  
result, err := orm.RegisterInserter[User](db).
    Insert(nil, &newUser).
    WithInvalidateCache().                   // 启用缓存失效
    WithInvalidateTags("user_list").         // 指定失效标签
    Exec(ctx)
```

### 手动失效

需要时可手动使缓存失效：

```go
// 使特定模型的缓存失效
err := db.InvalidateCache(ctx, "User", "user:123", "profile")

// 使用客户端API使缓存失效
err := client.InvalidateCache(ctx, "User", "user:123")
```

## 工作原理与流程

查询缓存的工作流程：

1. **查询拦截**：缓存中间件拦截查询请求
2. **缓存判断**：检查是否应该缓存该查询
3. **键生成**：生成唯一缓存键
4. **缓存查找**：尝试从缓存中获取结果
5. **缓存命中**：如果命中，直接返回缓存的结果
6. **缓存未命中**：执行实际查询，并将结果缓存
7. **返回结果**：返回查询结果给调用者