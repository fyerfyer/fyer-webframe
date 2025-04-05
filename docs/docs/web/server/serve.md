# Server

服务器提供了灵活的 HTTP 服务配置、优雅的启动和关闭机制以及丰富的配置选项。

## 基本配置

WebFrame 服务器以 `HTTPServer` 为核心，实现了 `Server` 接口，提供了完整的 HTTP 服务能力。

### 创建服务器

创建一个基本的 WebFrame 服务器非常简单：

```go
import "github.com/fyerfyer/fyer-webframe/web"

func main() {
    // 创建一个新的 HTTP 服务器
    server := web.NewHTTPServer()
    
    // 注册路由
    server.Get("/", func(ctx *web.Context) {
        ctx.String(200, "Hello, WebFrame!")
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

### 服务器接口

`Server` 接口定义了 WebFrame 服务器应具备的核心功能：

```go
type Server interface {
    http.Handler
    Start(addr string) error
    Shutdown(ctx context.Context) error
    
    // 路由注册方法
    Get(path string, handler HandlerFunc) RouteRegister
    Post(path string, handler HandlerFunc) RouteRegister
    Put(path string, handler HandlerFunc) RouteRegister
    Delete(path string, handler HandlerFunc) RouteRegister
    Patch(path string, handler HandlerFunc) RouteRegister
    Options(path string, handler HandlerFunc) RouteRegister
    
    // 路由组和中间件
    Group(prefix string) RouteGroup
    Middleware() MiddlewareManager
    
    // 模板引擎
    UseTemplate(tpl Template) Server
    GetTemplateEngine() Template
}
```

### 核心组件

服务器由以下核心组件组成：

1. **路由系统**: 处理 HTTP 请求路由和分发
2. **中间件链**: 处理请求前后的逻辑
3. **上下文管理**: 封装请求和响应操作
4. **模板引擎**: 提供页面渲染能力
5. **连接池管理**: 管理数据库、Redis 等连接资源

### 基础使用示例

```go
package main

import (
    "log"
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"
)

func main() {
    // 创建 HTTP 服务器
    server := web.NewHTTPServer()
    
    // 添加全局中间件
    server.Use("*", "*", recovery.Recovery())
    server.Use("*", "*", accesslog.NewMiddlewareBuilder().Build())
    
    // 注册根路由
    server.Get("/", func(ctx *web.Context) {
        ctx.String(200, "Welcome to WebFrame!")
    })
    
    // 注册 API 路由组
    api := server.Group("/api")
    
    // 添加用户相关路由
    api.Get("/users", listUsers)
    api.Post("/users", createUser)
    api.Get("/users/:id", getUserByID)
    
    // 启动服务器
    log.Println("Server starting on :8080")
    if err := server.Start(":8080"); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}

func listUsers(ctx *web.Context) {
    // 处理获取用户列表
    ctx.JSON(200, []map[string]any{
        {"id": 1, "name": "User 1"},
        {"id": 2, "name": "User 2"},
    })
}

func createUser(ctx *web.Context) {
    // 处理创建用户
    // ...
}

func getUserByID(ctx *web.Context) {
    // 获取路径参数
    id := ctx.PathParam("id").Value
    // ...
}
```

## 优雅关闭机制

服务器提供了优雅关闭机制，确保服务器关闭时能够正确处理现有请求，释放资源，防止连接泄漏。

### 实现原理

服务器的优雅关闭基于以下机制：

1. 使用 `context.Context` 控制关闭超时
2. 等待正在进行的请求处理完成
3. 关闭所有连接池资源
4. 退出服务器进程

### 使用方法

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/fyerfyer/fyer-webframe/web"
)

func main() {
    // 创建服务器
    server := web.NewHTTPServer()
    
    // 配置路由...
    
    // 创建通道监听系统信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    // 在后台启动服务器
    go func() {
        log.Println("Server starting on :8080")
        if err := server.Start(":8080"); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()
    
    // 等待退出信号
    <-quit
    log.Println("Shutting down server...")
    
    // 创建上下文，设置超时时间
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 优雅关闭服务器
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    
    log.Println("Server gracefully stopped")
}
```

### 资源释放流程

当调用 `Shutdown` 方法时，服务器会按以下顺序释放资源：

1. 停止接收新的连接请求
2. 等待所有活跃的请求处理完成
3. 关闭所有连接池资源
4. 释放服务器资源

```go
// Shutdown 优雅关闭
func (s *HTTPServer) Shutdown(ctx context.Context) error {
    s.start = false

    // 关闭连接池管理器
    if s.poolManager != nil {
        if err := s.poolManager.Shutdown(ctx); err != nil {
            return err
        }
    }

    return s.server.Shutdown(ctx)
}
```

## 选项模式

服务器采用选项模式进行配置，提供了灵活且易于扩展的配置方法。

### 什么是选项模式？

选项模式是一种函数式编程模式，通过定义一系列配置函数来设置对象的属性，避免使用大量的构造函数或复杂的构建器模式。

### WebFrame 中的选项模式

在 WebFrame 中，服务器选项通过 `ServerOption` 函数类型定义：

```go
// ServerOption 定义服务器选项
type ServerOption func(*HTTPServer)
```

### 可用选项

服务器提供以下内置选项：

#### 1. `WithReadTimeout` - 设置读取超时

```go
// 设置 10 秒读取超时
server := web.NewHTTPServer(web.WithReadTimeout(10 * time.Second))
```

#### 2. `WithWriteTimeout` - 设置写入超时

```go
// 设置 15 秒写入超时
server := web.NewHTTPServer(web.WithWriteTimeout(15 * time.Second))
```

#### 3. `WithTemplate` - 设置模板引擎

```go
// 创建模板引擎
tpl := web.NewGoTemplate(web.WithPattern("./templates/*.html"))

// 设置到服务器
server := web.NewHTTPServer(web.WithTemplate(tpl))
```

#### 4. `WithNotFoundHandler` - 自定义 404 处理器

```go
// 自定义 404 处理器
notFoundHandler := func(ctx *web.Context) {
    ctx.HTML(404, "<h1>page not found</h1><p>please check your URL</p>")
}

// 应用自定义处理器
server := web.NewHTTPServer(web.WithNotFoundHandler(notFoundHandler))
```

#### 5. `WithBasePath` - 设置基础路径前缀

```go
// 所有路由都将以 "/api/v1" 为前缀
server := web.NewHTTPServer(web.WithBasePath("/api/v1"))
```

#### 6. `WithPoolManager` - 设置连接池管理器

```go
// 创建连接池管理器
poolManager := myapp.NewPoolManager()

// 配置到服务器
server := web.NewHTTPServer(web.WithPoolManager(poolManager))
```

### 链式配置示例

选项可以组合使用，实现链式配置：

```go
server := web.NewHTTPServer(
    web.WithReadTimeout(10 * time.Second),
    web.WithWriteTimeout(15 * time.Second),
    web.WithBasePath("/api/v1"),
    web.WithTemplate(tpl),
    web.WithNotFoundHandler(customNotFoundHandler),
)
```

### 自定义选项

您可以根据需要创建自己的选项函数：

```go
// WithCustomLogger 自定义日志记录器选项
func WithCustomLogger(logger *log.Logger) web.ServerOption {
    return func(server *web.HTTPServer) {
        // 设置自定义日志记录器
        // 注意：需要在 HTTPServer 结构中添加相应的字段
    }
}

// 使用自定义选项
server := web.NewHTTPServer(WithCustomLogger(myCustomLogger))
```