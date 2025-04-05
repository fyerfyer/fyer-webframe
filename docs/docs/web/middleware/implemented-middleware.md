# Built-in Middleware

WebFrame 提供了一系列预先构建的中间件，用于处理常见的 Web 应用需求，如日志记录、错误恢复、性能监控和分布式追踪。

## 访问日志中间件

访问日志中间件记录所有 HTTP 请求的详细信息，对于调试和监控应用程序非常有用。

### 功能特点

- 记录请求主机名、路由、路径和 HTTP 方法
- 支持自定义日志格式和输出目标
- 使用构建器模式提供灵活配置

### 使用方法

基本用法：

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
)

func main() {
    server := web.NewHTTPServer()
    
    // 使用默认配置
    server.Use("*", "/*", accesslog.NewMiddlewareBuilder().Build())
    
    // 启动服务器
    server.Start(":8080")
}
```

自定义日志记录器：

```go
import (
    "log"
    "os"
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
)

func main() {
    server := web.NewHTTPServer()
    
    // 创建自定义日志记录器
    logger := log.New(os.Stdout, "[ACCESS] ", log.LstdFlags)
    
    // 构建访问日志中间件
    accessLogMiddleware := accesslog.NewMiddlewareBuilder().
        SetLogger(func(content string) {
            logger.Println(content)
        }).
        Build()
    
    // 注册中间件
    server.Use("*", "/*", accessLogMiddleware)
    
    // 启动服务器
    server.Start(":8080")
}
```

### 输出样例

访问日志中间件输出 JSON 格式的日志条目，包含以下字段：

```json
{"Host":"localhost:8080","Route":"/users/:id","Path":"/users/123","HttpMethod":"GET"}
```

## 恢复处理（Recovery）中间件

恢复中间件捕获请求处理过程中发生的 panic，防止应用程序崩溃，并将其转换为 500 内部服务器错误响应。

### 功能特点

- 自动捕获并恢复所有 panic
- 记录详细的堆栈跟踪信息
- 向客户端返回友好的错误消息
- 保持服务的稳定性和可用性

### 使用方法

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"
)

func main() {
    server := web.NewHTTPServer()
    
    // 注册恢复中间件（通常应该是第一个中间件）
    server.Use("*", "/*", recovery.Recovery())
    
    // 注册可能会 panic 的路由
    server.Get("/risky", func(ctx *web.Context) {
        // 这将触发 panic
        var p *int = nil
        *p = 1 // nil 指针解引用
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

### 行为示例

当请求处理中发生 panic 时：

1. 恢复中间件捕获 panic 并记录详细的堆栈跟踪
2. 控制台会打印类似以下内容：

```
PANIC: runtime error: invalid memory address or nil pointer dereference
goroutine 18 [running]:
runtime/debug.Stack()
    /usr/local/go/src/runtime/debug/stack.go:24 +0x65
github.com/fyerfyer/fyer-webframe/web/middleware/recovery.Recovery.func1.1()
    /path/to/recovery/recovery.go:15 +0x75
...
```

3. 客户端会收到 500 状态码和一个包含错误 ID 的 JSON 响应：

```json
{"error": "Internal server error occurred. Error ID: runtime error: invalid memory address or nil pointer dereference"}
```

## Prometheus 监控中间件

Prometheus 中间件收集 HTTP 请求的性能指标，并以 Prometheus 格式导出，便于与 Prometheus 监控系统集成。

### 功能特点

- 收集请求处理时间的分位数统计（0.5, 0.9, 0.99, 0.999）
- 按请求方法、路径和响应状态码分类指标
- 支持自定义指标名称和标签

### 使用方法

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    server := web.NewHTTPServer()
    
    // 构建 Prometheus 中间件
    metricsMiddleware := &prometheus.MiddlewareBuilder{
        NameSpace: "webframe",
        Name:      "http_requests",
        SubSystem: "web",
        Help:      "HTTP request duration in microseconds",
    }.Build()
    
    // 注册中间件
    server.Use("*", "/*", metricsMiddleware)
    
    // 添加 Prometheus 指标端点
    server.Get("/metrics", func(ctx *web.Context) {
        promhttp.Handler().ServeHTTP(ctx.Resp, ctx.Req)
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

### 指标样例

访问 `/metrics` 端点可以获取类似以下的 Prometheus 格式指标：

```
# HELP webframe_web_http_requests HTTP request duration in microseconds
# TYPE webframe_web_http_requests summary
webframe_web_http_requests{method="GET",path="/users",status="200",quantile="0.5"} 125
webframe_web_http_requests{method="GET",path="/users",status="200",quantile="0.9"} 220
webframe_web_http_requests{method="GET",path="/users",status="200",quantile="0.99"} 350
webframe_web_http_requests_sum{method="GET",path="/users",status="200"} 5342.05
webframe_web_http_requests_count{method="GET",path="/users",status="200"} 42
```

## 链路追踪中间件

链路追踪中间件实现了 OpenTelemetry 分布式追踪协议，可以帮助开发者理解和分析复杂系统中请求的流转路径和性能瓶颈。

### 功能特点

- 与 OpenTelemetry 规范兼容
- 自动提取和传播追踪上下文
- 记录 HTTP 请求的关键属性
- 支持自定义 tracer 配置

### 使用方法

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/opentracing"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "log"
)

func main() {
    // 设置 OpenTelemetry 导出器（以 Jaeger 为例）
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))
    if err != nil {
        log.Fatalf("Failed to create Jaeger exporter: %v", err)
    }
    
    // 创建 TraceProvider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithSampler(trace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)
    
    // 创建 Web 服务器
    server := web.NewHTTPServer()
    
    // 构建并注册链路追踪中间件
    tracingMiddleware := &opentracing.MiddlewareBuilder{}.Build()
    server.Use("*", "/*", tracingMiddleware)
    
    // 注册路由
    server.Get("/users/:id", func(ctx *web.Context) {
        // 处理请求...
        ctx.JSON(200, map[string]interface{}{
            "id": ctx.PathParam("id").Value,
            "name": "User Name",
        })
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

### 追踪数据示例

追踪数据将包含以下属性：

- `http.method` - HTTP 方法（GET, POST 等）
- `http.host` - 主机名
- `http.url` - 完整 URL
- `http.scheme` - URL 方案（http, https）
- `span.kind` - 跨度类型（server）
- `component` - 组件名称（web）
- `http.proto` - HTTP 协议版本

这些数据可以在 Jaeger UI 或其他 OpenTelemetry 兼容的可视化工具中查看。

## 会话（Session）中间件

会话中间件为 Web 应用提供会话管理功能，支持会话创建、检索和刷新。

### 功能特点

- 自动创建会话（可配置）
- 支持自定义会话初始化逻辑
- 会话自动刷新
- 与多种会话存储后端兼容

### 使用方法

基本用法：

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/session"
    "github.com/fyerfyer/fyer-webframe/web/session/cookiepropagator"
    "github.com/fyerfyer/fyer-webframe/web/session/redissession"
    "time"
)

func main() {
    server := web.NewHTTPServer()
    
    // 创建会话存储
    storage, err := redissession.NewRedisStorage(
        "localhost:6379",
        "",
        0,
        time.Hour,
    )
    if err != nil {
        panic(err)
    }
    
    // 创建会话管理器
    cookieProp := cookiepropagator.NewPropagator("session_id")
    sessionManager := session.NewManager(storage, cookieProp)
    
    // 创建并注册会话中间件
    sessionMiddleware := session.NewSessionMiddleware(sessionManager, true)
    server.Use("*", "/*", sessionMiddleware.Build())
    
    // 路由处理
    server.Get("/profile", func(ctx *web.Context) {
        sess, err := sessionManager.GetSession(ctx)
        if err != nil {
            ctx.JSON(401, map[string]string{"error": "Unauthorized"})
            return
        }
        
        username, err := sess.Get("username")
        if err != nil {
            ctx.JSON(401, map[string]string{"error": "Not logged in"})
            return
        }
        
        ctx.JSON(200, map[string]string{"username": username.(string)})
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

添加自定义会话初始化器：

```go
// 创建带初始化器的会话中间件
sessionMiddleware := session.NewSessionMiddleware(sessionManager, true).
    WithInitializer(func(s session.Session) error {
        // 设置默认会话值
        err := s.Set("created_at", time.Now().Format(time.RFC3339))
        if err != nil {
            return err
        }
        
        err = s.Set("visits", 1)
        if err != nil {
            return err
        }
        
        return nil
    })

// 注册中间件
server.Use("*", "/*", sessionMiddleware.Build())
```

## 组合使用内置中间件

以下是结合多个内置中间件的完整示例：

```go
import (
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
    "github.com/fyerfyer/fyer-webframe/web/middleware/opentracing"
    "github.com/fyerfyer/fyer-webframe/web/middleware/prometheus"
    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"
    "github.com/fyerfyer/fyer-webframe/web/middleware/session"
    "log"
)

func main() {
    server := web.NewHTTPServer()
    
    // 1. 恢复中间件 (最外层，捕获所有 panic)
    server.Use("*", "/*", recovery.Recovery())
    
    // 2. 访问日志中间件
    accessLogMiddleware := accesslog.NewMiddlewareBuilder().
        SetLogger(func(content string) {
            log.Println(content)
        }).
        Build()
    server.Use("*", "/*", accessLogMiddleware)
    
    // 3. 监控中间件
    metricsMiddleware := &prometheus.MiddlewareBuilder{
        NameSpace: "webframe",
        Name:      "http_requests",
        SubSystem: "web",
        Help:      "HTTP request duration in microseconds",
    }.Build()
    server.Use("*", "/*", metricsMiddleware)
    
    // 4. 链路追踪中间件
    tracingMiddleware := &opentracing.MiddlewareBuilder{}.Build()
    server.Use("*", "/*", tracingMiddleware)
    
    // 5. 会话中间件 (最内层，靠近业务逻辑)
    // 将会话中间件应用于需要认证的路由
    sessionMiddleware := setupSessionMiddleware() // 详细实现省略
    server.Use("GET", "/api/users/*", sessionMiddleware.Build())
    server.Use("POST", "/api/users/*", sessionMiddleware.Build())
    
    // 注册路由...
    
    // 启动服务器
    server.Start(":8080")
}
```
