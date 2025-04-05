# Middleware

中间件是 WebFrame 框架的核心功能之一，它提供了一种强大而灵活的机制，允许您在请求处理流程中插入自定义逻辑。

### WebFrame 中间件的核心概念

在 WebFrame 中，中间件被定义为以下函数类型：

```go
// HandlerFunc 定义请求处理函数
type HandlerFunc func(ctx *Context)

// Middleware 定义中间件函数
type Middleware func(HandlerFunc) HandlerFunc
```

这种洋葱模型设计使中间件可以在调用下一个处理函数前和后执行代码，形成一个围绕实际处理器的处理链。

### 中间件类型

WebFrame 支持多种类型的中间件，根据其注册路径的不同分为：

1. **静态中间件**：匹配确切的路径，如 users
2. **参数中间件**：匹配带参数的路径，如 `/users/:id`
3. **正则中间件**：匹配带正则表达式的路径，如 `/users/:id([0-9]+)`
4. **通配符中间件**：匹配通配符路径，如 `/users/*`
5. **全局中间件**：匹配所有路径，通常注册为 `/*`

### 中间件执行顺序

中间件执行遵循以下规则：

1. **按匹配特定性**：静态路径 > 正则路径 > 参数路径 > 通配符路径
2. **按注册顺序**：先注册的中间件先执行
3. **洋葱模型执行**：请求阶段从外到内，响应阶段从内到外

举例说明洋葱模型执行顺序：

```go
s.Use("GET", "/path", func(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        // 请求阶段 (1)
        next(ctx)
        // 响应阶段 (6)
    }
})

s.Use("GET", "/path", func(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        // 请求阶段 (2)
        next(ctx)
        // 响应阶段 (5)
    }
})

s.Get("/path", func(ctx *Context) {
    // 处理请求 (3)
    // 构建响应 (4)
})
```

执行顺序是：(1) → (2) → (3) → (4) → (5) → (6)

## 使用中间件

### 注册中间件

WebFrame 提供了多种注册中间件的方式：

#### 1. 使用 `Use` 方法

```go
// 全局中间件
s.Use("*", "/*", loggerMiddleware)

// 特定 HTTP 方法的中间件
s.Use("GET", "/api/*", authMiddleware)

// 特定路径的中间件
s.Use("POST", "/users/:id", validateUserMiddleware)
```

#### 2. 使用中间件管理器

```go
// 全局中间件
s.Middleware().Global().Add(recoveryMiddleware, loggerMiddleware)

// 特定路径中间件
s.Middleware().For("GET", "/api/users/*").Add(authMiddleware)

// 条件中间件（只有满足条件时才执行）
s.Middleware().When(func(c *Context) bool {
    return c.GetHeader("X-API-Version") == "v2"
}).Add(apiVersionMiddleware)
```

#### 3. 使用路由链式 API

```go
s.Get("/admin/dashboard", dashboardHandler).
    Middleware(authMiddleware, adminCheckMiddleware)
```

#### 4. 使用路由组中间件

```go
// 创建带中间件的路由组
api := s.Group("/api").Use(
    authMiddleware,
    rateLimitMiddleware,
)

// 该组中的所有路由都会应用上述中间件
api.Get("/users", listUsers)
api.Post("/users", createUser)
```

### 中间件流程控制

WebFrame 提供了以下控制中间件执行流程的方法：

#### 1. 终止请求处理（Abort）

```go
func authMiddleware(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        token := ctx.GetHeader("Authorization")
        if token == "" {
            ctx.Abort()  // 终止后续中间件和处理器执行
            ctx.JSON(401, map[string]string{"error": "Unauthorized"})
            return
        }
        next(ctx)
    }
}
```

#### 2. 检查请求是否已终止

```go
func loggingMiddleware(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        // 请求阶段
        next(ctx)
        // 响应阶段
        if !ctx.IsAborted() {
            // 请求未被终止时才执行日志记录
            log.Printf("Request completed with status: %d", ctx.RespStatusCode)
        }
    }
}
```

#### 3. 显式调用下一个处理器

```go
func timingMiddleware(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        start := time.Now()
        
        ctx.Next(next)  // 显式调用下一个处理器
        
        duration := time.Since(start)
        log.Printf("Request took %v", duration)
    }
}
```

### 中间件共享数据

您可以使用 `Context.UserValues` 在中间件和处理器之间共享数据：

```go
func userLoaderMiddleware(next HandlerFunc) HandlerFunc {
    return func(ctx *Context) {
        userID := ctx.PathInt("id").Value
        user, err := getUserFromDB(userID)
        if err != nil {
            ctx.JSON(500, map[string]string{"error": "Failed to load user"})
            ctx.Abort()
            return
        }
        
        // 将用户数据存储在上下文中
        ctx.UserValues["user"] = user
        next(ctx)
    }
}

// 处理器可以访问中间件存储的数据
func getUserProfile(ctx *Context) {
    user, ok := ctx.UserValues["user"].(User)
    if !ok {
        ctx.JSON(500, map[string]string{"error": "User not found in context"})
        return
    }
    
    ctx.JSON(200, user.Profile)
}
```

## 编写自定义中间件

### 基本结构

自定义中间件遵循以下基本结构：

```go
func MyMiddleware(next web.HandlerFunc) web.HandlerFunc {
    // 初始化阶段：在服务器启动时执行一次
    
    return func(ctx *web.Context) {
        // 请求阶段：在调用下一个处理器前执行
        
        next(ctx)  // 调用下一个处理器或中间件
        
        // 响应阶段：在下一个处理器返回后执行
    }
}
```

### 带配置的中间件

对于需要配置选项的中间件，通常采用闭包或构建器模式：

```go
// 选项模式
type LoggerOptions struct {
    Format      string
    LogRequest  bool
    LogResponse bool
}

func NewLoggerMiddleware(options LoggerOptions) web.Middleware {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            if options.LogRequest {
                log.Printf("Request: %s %s", ctx.Req.Method, ctx.Req.URL.Path)
            }
            
            next(ctx)
            
            if options.LogResponse {
                log.Printf("Response: %d (%s %s)", 
                    ctx.RespStatusCode, ctx.Req.Method, ctx.Req.URL.Path)
            }
        }
    }
}
```

使用：

```go
s.Use("*", "/*", NewLoggerMiddleware(LoggerOptions{
    Format:      "standard",
    LogRequest:  true,
    LogResponse: true,
}))
```

### 构建器模式

更复杂的中间件可以使用构建器模式：

```go
// 构建器模式
type RateLimiterBuilder struct {
    limit        int
    windowSeconds int
    keyFunc      func(*web.Context) string
}

func NewRateLimiterBuilder() *RateLimiterBuilder {
    return &RateLimiterBuilder{
        limit:        100,
        windowSeconds: 60,
        keyFunc:      defaultKeyFunc,
    }
}

func (b *RateLimiterBuilder) WithLimit(limit int) *RateLimiterBuilder {
    b.limit = limit
    return b
}

func (b *RateLimiterBuilder) WithWindow(seconds int) *RateLimiterBuilder {
    b.windowSeconds = seconds
    return b
}

func (b *RateLimiterBuilder) WithKeyFunc(fn func(*web.Context) string) *RateLimiterBuilder {
    b.keyFunc = fn
    return b
}

func (b *RateLimiterBuilder) Build() web.Middleware {
    // 创建速率限制器
    limiter := createRateLimiter(b.limit, b.windowSeconds)
    
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            key := b.keyFunc(ctx)
            if !limiter.Allow(key) {
                ctx.JSON(429, map[string]string{"error": "Too many requests"})
                ctx.Abort()
                return
            }
            next(ctx)
        }
    }
}
```

使用：

```go
limiter := NewRateLimiterBuilder().
    WithLimit(200).
    WithWindow(30).
    WithKeyFunc(func(ctx *web.Context) string {
        return ctx.ClientIP()
    }).
    Build()

s.Use("*", "/*", limiter)
```

### 中间件示例

以下是几个常用中间件的实现示例：

#### 1. 请求日志中间件

```go
func RequestLoggerMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        start := time.Now()
        method := ctx.Req.Method
        path := ctx.Req.URL.Path
        
        log.Printf("Request started: %s %s", method, path)
        
        next(ctx)
        
        duration := time.Since(start)
        log.Printf("Request completed: %s %s %d %v", 
            method, path, ctx.RespStatusCode, duration)
    }
}
```

#### 2. 错误恢复中间件

```go
func RecoveryMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v", err)
                debug.PrintStack()
                
                ctx.JSON(500, map[string]string{
                    "error": "Internal server error",
                })
            }
        }()
        
        next(ctx)
    }
}
```

#### 3. 认证中间件

```go
func AuthMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        token := ctx.GetHeader("Authorization")
        
        // 简单的 Bearer token 检查
        if !strings.HasPrefix(token, "Bearer ") {
            ctx.JSON(401, map[string]string{"error": "Invalid authorization format"})
            ctx.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(token, "Bearer ")
        
        user, err := validateToken(tokenString)
        if err != nil {
            ctx.JSON(401, map[string]string{"error": "Invalid token"})
            ctx.Abort()
            return
        }
        
        // 将用户信息存储在上下文中
        ctx.UserValues["user"] = user
        ctx.UserValues["userID"] = user.ID
        
        next(ctx)
    }
}

func validateToken(token string) (*User, error) {
    // 实现 token 验证逻辑
    // ...
}
```

#### 4. CORS 中间件

```go
func CORSMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        ctx.SetHeader("Access-Control-Allow-Origin", "*")
        ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        ctx.SetHeader("Access-Control-Allow-Headers", 
            "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization")
        
        // 处理预检请求
        if ctx.Req.Method == "OPTIONS" {
            ctx.Status(204)
            return
        }
        
        next(ctx)
    }
}
```

#### 5. 请求 ID 中间件

```go
func RequestIDMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        requestID := ctx.GetHeader("X-Request-ID")
        
        // 如果请求没有 ID，生成一个
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        // 将请求 ID 添加到响应头
        ctx.SetHeader("X-Request-ID", requestID)
        
        // 存储在上下文中供后续使用
        ctx.UserValues["requestID"] = requestID
        
        next(ctx)
    }
}
```

## 完整示例

以下是一个综合使用多种中间件的完整示例：

```go
package main

import (
    "log"
    "time"
    
    "github.com/fyerfyer/fyer-webframe/web"
)

func main() {
    server := web.NewHTTPServer()
    
    // 1. 全局恢复中间件
    server.Middleware().Global().Add(func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            defer func() {
                if err := recover(); err != nil {
                    log.Printf("Recovered from panic: %v", err)
                    ctx.JSON(500, map[string]string{"error": "Internal server error"})
                }
            }()
            next(ctx)
        }
    })
    
    // 2. 请求日志中间件
    server.Middleware().Global().Add(func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            start := time.Now()
            path := ctx.Req.URL.Path
            method := ctx.Req.Method
            
            log.Printf("Request: %s %s", method, path)
            next(ctx)
            
            duration := time.Since(start)
            log.Printf("Response: %s %s %d %v", method, path, ctx.RespStatusCode, duration)
        }
    })
    
    // 3. API 路径特定的中间件
    apiMiddleware := func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            log.Printf("API request received: %s", ctx.Req.URL.Path)
            ctx.SetHeader("X-API-Version", "1.0")
            next(ctx)
        }
    }
    
    // 4. 认证中间件
    authMiddleware := func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            token := ctx.GetHeader("Authorization")
            if token == "" {
                ctx.JSON(401, map[string]string{"error": "Authentication required"})
                ctx.Abort()
                return
            }
            
            // 在实际应用中验证token
            ctx.UserValues["userID"] = 123
            next(ctx)
        }
    }
    
    // 5. 管理员检查中间件
    adminMiddleware := func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            userID, ok := ctx.UserValues["userID"].(int)
            if !ok || !isAdmin(userID) {
                ctx.JSON(403, map[string]string{"error": "Admin access required"})
                ctx.Abort()
                return
            }
            next(ctx)
        }
    }
    
    // 创建API路由组并应用中间件
    api := server.Group("/api").Use(apiMiddleware)
    
    // 公开API端点
    api.Get("/public", func(ctx *web.Context) {
        ctx.JSON(200, map[string]string{"message": "Public API"})
    })
    
    // 需要认证的API端点
    protected := api.Group("/protected").Use(authMiddleware)
    protected.Get("/user", func(ctx *web.Context) {
        userID := ctx.UserValues["userID"].(int)
        ctx.JSON(200, map[string]interface{}{
            "user_id": userID,
            "message": "Protected user data",
        })
    })
    
    // 需要管理员权限的API端点
    admin := protected.Group("/admin").Use(adminMiddleware)
    admin.Get("/dashboard", func(ctx *web.Context) {
        ctx.JSON(200, map[string]string{"message": "Admin dashboard"})
    })
    
    log.Println("Server starting on :8080")
    server.Start(":8080")
}

func isAdmin(userID int) bool {
    // 在实际应用中检查用户是否为管理员
    admins := []int{123, 456}
    for _, id := range admins {
        if id == userID {
            return true
        }
    }
    return false
}
```