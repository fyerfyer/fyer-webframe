# Session

WebFrame 框架提供了完备的会话管理系统，支持多种存储后端和会话传播方式。

## 会话基础

### 会话管理架构

WebFrame 的会话管理系统采用模块化设计，由三个核心组件组成：

1. **Session**：表示单个会话，负责具体数据的存取操作
2. **Storage**：负责会话的生命周期管理（创建、查找、刷新和删除）
3. **Propagator**：负责会话 ID 在请求和响应之间的传递

这种设计使得会话管理系统具有很高的灵活性和可扩展性，可以方便地替换或自定义各个组件。

### 核心接口

会话管理的核心接口定义如下：

```go
// Session 负责会话具体数据的操作
type Session interface {
    Get(ctx context.Context, key string) (any, error)
    Set(ctx context.Context, key string, value any) error
    ID() string
    Touch(ctx context.Context) error
}

// Storage 负责会话的生命周期管理
type Storage interface {
    Create(ctx context.Context, id string) (Session, error)
    Refresh(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (Session, error)
    Delete(ctx context.Context, id string) error
    GC(ctx context.Context) error
    Close() error
}

// Propagator 负责会话ID在请求和响应中的传递
type Propagator interface {
    Insert(id string, resp http.ResponseWriter) error
    Extract(req *http.Request) (string, error)
    Remove(resp http.ResponseWriter) error
}
```

### 会话管理器

会话管理器（Manager）整合了 Storage 和 Propagator，为应用程序提供了便于使用的会话管理方法调用：

```go
type Manager struct {
    Storage
    Propagator
    sessionKey string
}

func NewMagager(storage Storage, propagator Propagator, sessionKey string) *Manager {
    return &Manager{
        Storage:    storage,
        Propagator: propagator,
        sessionKey: sessionKey,
    }
}
```

会话管理器提供了以下主要方法：

- **InitSession**：初始化一个新会话并将其注入到请求上下文中
- **GetSession**：从请求上下文或传播器中获取会话
- **RefreshSession**：刷新会话的过期时间
- **TouchSession**：更新会话过期时间但不改变其数据
- **DeleteSession**：删除会话

### 会话中间件

WebFrame 提供了会话中间件，用于自动处理请求中的会话：

```go
// Middleware 用于初始化和处理会话的中间件
type Middleware struct {
    SessionManager *session.Manager
    AutoCreate     bool
    Initializer    SessionInitializer
}

// SessionInitializer 初始化最初的会话值
type SessionInitializer func(s session.Session) error
```

会话中间件可以配置为：

- 自动创建新会话（当请求没有会话时）
- 使用自定义的会话初始化器来设置会话的初始数据
- 自动刷新会话过期时间

## Cookie 会话

Cookie 是在 Web 应用程序中传递会话 ID 的最常用方法。WebFrame 提供了 `CookiePropagator` 实现，用于通过 HTTP Cookie 机制传递会话 ID。

### CookiePropagator 基本用法

```go
import "github.com/fyerfyer/fyer-webframe/web/session/cookiepropagator"

// 创建 Cookie 传播器
propagator := cookiepropagator.NewCookiePropagator()

// 或者使用选项配置
propagator := cookiepropagator.NewCookiePropagator(
    cookiepropagator.WithCookieName("my_session"),
    cookiepropagator.WithCookiePath("/"),
    cookiepropagator.WithCookieMaxAge(3600),  // 1小时
    cookiepropagator.WithCookieSecure(true),
    cookiepropagator.WithCookieHTTPOnly(true),
    cookiepropagator.WithSameSite(http.SameSiteStrictMode),
)
```

### 配置选项

`CookiePropagator` 支持以下配置选项：

```go
// 设置 cookie 名称
WithCookieName(name string)

// 设置 cookie 路径
WithCookiePath(path string)

// 设置 cookie 域
WithCookieDomain(domain string)

// 设置 cookie 最大存活时间（秒）
WithCookieMaxAge(maxAge int)

// 设置 cookie 安全标志
WithCookieSecure(secure bool)

// 设置 cookie HTTP only 标志
WithCookieHTTPOnly(httpOnly bool)

// 设置 cookie SameSite 属性
WithSameSite(sameSite http.SameSite)
```

### 安全最佳实践

为了确保会话安全，建议采用以下 Cookie 配置：

```go
propagator := cookiepropagator.NewCookiePropagator(
    cookiepropagator.WithCookieName("session_id"),
    cookiepropagator.WithCookiePath("/"),
    cookiepropagator.WithCookieHTTPOnly(true),  // 防止 JavaScript 访问
    cookiepropagator.WithCookieSecure(true),    // 仅通过 HTTPS 发送
    cookiepropagator.WithSameSite(http.SameSiteStrictMode),  // 防止 CSRF
)
```

## Redis 会话存储

Redis 是一种常用的会话存储后端，适用于分布式环境和需要高性能会话管理的场景。WebFrame 提供了基于 Redis 的会话存储实现，使用连接池管理 Redis 连接。

### Redis 会话存储设置

```go
import (
    "github.com/fyerfyer/fyer-kit/pool"
    "github.com/fyerfyer/fyer-webframe/web/session/redissession"
    "github.com/go-redis/redis/v8"
    "time"
)

// 创建 Redis 连接池
redisPool, err := pool.NewPool(
    func(ctx context.Context) (interface{}, error) {
        client := redis.NewClient(&redis.Options{
            Addr:     "localhost:6379",
            Password: "",
            DB:       0,
        })
        return client, nil
    },
    func(ctx context.Context, conn interface{}) error {
        client := conn.(*redis.Client)
        return client.Close()
    },
    pool.WithMaxActive(100),
    pool.WithMaxIdle(10),
    pool.WithIdleTimeout(time.Minute),
)
if err != nil {
    log.Fatalf("Failed to create Redis pool: %v", err)
}

// 创建 Redis 会话存储
storage := redissession.NewRedisStorage(
    redisPool,
    redissession.WithExpireTime(time.Hour), // 会话过期时间
    redissession.WithPrefix("sess_"),       // 会话键前缀
    redissession.WithCleanupInterval(time.Minute*5), // 过期会话清理间隔
)
```

### Redis 会话存储配置

`RedisStorage` 支持以下配置选项：

```go
// 设置会话过期时间
WithExpireTime(expireTime time.Duration)

// 设置会话键前缀
WithPrefix(prefix string)

// 设置过期会话清理间隔
WithCleanupInterval(interval time.Duration)
```

### Redis 会话存储特性

1. **连接池管理**：使用连接池优化 Redis 连接管理，提高性能和资源利用率
2. **本地缓存**：会话数据在本地缓存，减少对 Redis 的请求
3. **自动清理**：后台任务定期清理过期会话
4. **并发安全**：使用互斥锁保证会话操作的线程安全
5. **惰性加载**：会话数据按需从 Redis 加载

## 完整示例

以下示例展示了如何在 WebFrame 应用中集成会话管理：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/fyerfyer/fyer-kit/pool"
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/session"
    "github.com/fyerfyer/fyer-webframe/web/session/cookiepropagator"
    "github.com/fyerfyer/fyer-webframe/web/session/redissession"
    "github.com/go-redis/redis/v8"
)

func main() {
    // 创建 HTTP 服务器
    server := web.NewHTTPServer()

    // 创建 Redis 连接池
    redisPool, err := pool.NewPool(
        func(ctx context.Context) (interface{}, error) {
            client := redis.NewClient(&redis.Options{
                Addr:     "localhost:6379",
                Password: "",
                DB:       0,
            })
            return client, nil
        },
        func(ctx context.Context, conn interface{}) error {
            client := conn.(*redis.Client)
            return client.Close()
        },
        pool.WithMaxActive(100),
        pool.WithMaxIdle(10),
    )
    if err != nil {
        log.Fatalf("Failed to create Redis pool: %v", err)
    }

    // 创建 Redis 会话存储
    storage := redissession.NewRedisStorage(
        redisPool,
        redissession.WithExpireTime(time.Hour),
        redissession.WithPrefix("sess_"),
    )

    // 创建 Cookie 会话传播器
    propagator := cookiepropagator.NewCookiePropagator(
        cookiepropagator.WithCookieName("session_id"),
        cookiepropagator.WithCookieHTTPOnly(true),
        cookiepropagator.WithCookieMaxAge(3600),
    )

    // 创建会话管理器
    sessionManager := session.NewMagager(storage, propagator, "session")

    // 创建会话中间件
    sessionMiddleware := session.NewSessionMiddleware(sessionManager, true).
        WithInitializer(func(s session.Session) error {
            // 设置会话初始值
            ctx := context.Background()
            err := s.Set(ctx, "created_at", time.Now().Format(time.RFC3339))
            if err != nil {
                return err
            }
            
            err = s.Set(ctx, "visits", 1)
            if err != nil {
                return err
            }
            
            return nil
        })

    // 注册会话中间件
    server.Use("*", "/*", sessionMiddleware.Build())

    // 注册路由处理程序
    server.Get("/", func(ctx *web.Context) {
        sess, err := sessionManager.GetSession(ctx)
        if err != nil {
            ctx.String(500, "Failed to get session: %v", err)
            return
        }

        bgCtx := context.Background()
        
        // 获取访问次数
        visitsVal, err := sess.Get(bgCtx, "visits")
        if err != nil {
            ctx.String(500, "Failed to get visits: %v", err)
            return
        }
        
        visits, ok := visitsVal.(float64)
        if !ok {
            visits = 0
        }
        
        // 增加访问次数
        visits++
        err = sess.Set(bgCtx, "visits", visits)
        if err != nil {
            ctx.String(500, "Failed to update visits: %v", err)
            return
        }

        // 获取创建时间
        createdAt, _ := sess.Get(bgCtx, "created_at")
        
        ctx.JSON(200, map[string]interface{}{
            "session_id": sess.ID(),
            "visits":     visits,
            "created_at": createdAt,
        })
    })

    // 登录示例
    server.Post("/login", func(ctx *web.Context) {
        // 获取用户凭据
        username := ctx.FormValue("username").Value
        password := ctx.FormValue("password").Value
        
        // 验证凭据 (示例)
        if username == "admin" && password == "password" {
            // 获取会话
            sess, err := sessionManager.GetSession(ctx)
            if err != nil {
                ctx.String(500, "Failed to get session: %v", err)
                return
            }
            
            // 设置用户信息到会话
            bgCtx := context.Background()
            err = sess.Set(bgCtx, "user_id", 1)
            if err != nil {
                ctx.String(500, "Failed to set user_id: %v", err)
                return
            }
            
            err = sess.Set(bgCtx, "username", username)
            if err != nil {
                ctx.String(500, "Failed to set username: %v", err)
                return
            }
            
            err = sess.Set(bgCtx, "logged_in_at", time.Now().Format(time.RFC3339))
            if err != nil {
                ctx.String(500, "Failed to set logged_in_at: %v", err)
                return
            }
            
            ctx.JSON(200, map[string]string{
                "message": "Login successful",
            })
        } else {
            ctx.JSON(401, map[string]string{
                "error": "Invalid credentials",
            })
        }
    })

    // 登出示例
    server.Post("/logout", func(ctx *web.Context) {
        err := sessionManager.DeleteSession(ctx)
        if err != nil {
            ctx.String(500, "Failed to delete session: %v", err)
            return
        }
        
        ctx.JSON(200, map[string]string{
            "message": "Logout successful",
        })
    })

    // 受保护的 API 示例
    server.Get("/profile", func(ctx *web.Context) {
        sess, err := sessionManager.GetSession(ctx)
        if err != nil {
            ctx.JSON(401, map[string]string{
                "error": "Authentication required",
            })
            return
        }
        
        bgCtx := context.Background()
        userID, err := sess.Get(bgCtx, "user_id")
        if err != nil || userID == nil {
            ctx.JSON(401, map[string]string{
                "error": "Authentication required",
            })
            return
        }
        
        // 获取用户个人资料
        username, _ := sess.Get(bgCtx, "username")
        
        ctx.JSON(200, map[string]interface{}{
            "user_id":  userID,
            "username": username,
            "email":    fmt.Sprintf("%s@example.com", username),
        })
    })

    // 启动服务器
    log.Println("Server starting on :8080")
    err = server.Start(":8080")
    if err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}
```