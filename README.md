# fyer-webFrame

WebFrame 是一个高性能、功能丰富的 Go Web 框架，提供完整的 Web 应用开发解决方案，包括路由管理、中间件系统、ORM、模板引擎和请求处理等核心功能。

## 安装

```bash
# 安装框架
go get github.com/fyerfyer/fyer-webframe

# 安装脚手架工具
go install github.com/fyerfyer/fyer-webframe/cmd/scaffold@latest
```

## 快速开始

使用脚手架创建新项目：

```bash
# 创建新项目
scaffold -name myproject

# 进入项目目录
cd myproject

# 运行项目
go run .
```

访问 http://localhost:8080 查看默认的欢迎页面。

## 基本用法

### 创建服务器和路由

```go
package main

import (
    "github.com/fyerfyer/fyer-webframe/web"
    "net/http"
)

func main() {
    // 创建 HTTP 服务器
    server := web.NewHTTPServer()
    
    // 注册路由
    server.Get("/", func(ctx *web.Context) {
        ctx.String(http.StatusOK, "Hello, WebFrame!")
    })
    
    // 注册带参数的路由
    server.Get("/users/:id", func(ctx *web.Context) {
        id := ctx.PathParam("id").Value
        ctx.JSON(http.StatusOK, map[string]string{
            "id": id,
            "name": "User " + id,
        })
    })
    
    // 启动服务器
    server.Start(":8080")
}
```

### 使用中间件

```go
// 创建日志中间件
logMiddleware := func(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        // 处理请求前的逻辑
        start := time.Now()
        
        // 调用下一个处理器
        next(ctx)
        
        // 处理请求后的逻辑
        duration := time.Since(start)
        fmt.Printf("[%s] %s - %d - %v\n", 
            ctx.Req.Method, ctx.Req.URL.Path, ctx.RespStatusCode, duration)
    }
}

// 全局中间件
server.Middleware().Global().Add(logMiddleware)

// 路径特定中间件
server.Middleware().For("GET", "/api/*").Add(logMiddleware)
```

### 使用路由组

```go
// 创建API路由组
api := server.Group("/api")

// 用户路由组
users := api.Group("/users")
users.Get("", listUsers)
users.Get("/:id", getUserById)
users.Post("", createUser)
users.Put("/:id", updateUser)
users.Delete("/:id", deleteUser)

// 嵌套路由组
admin := api.Group("/admin")
admin.Get("/stats", getStats)
```

## 性能

WebFrame 各模块的基准测试结果如下：

### 路由性能

```
BenchmarkRouter_StaticRoutes-4          1,698,499    797.6 ns/op
BenchmarkRouter_ParamRoutes-4             696,019    1543 ns/op
BenchmarkRouter_WildcardRoutes-4        1,365,712    876.0 ns/op
BenchmarkRouter_RegexRoutes-4             715,665    1786 ns/op
```

### 中间件性能

```
BenchmarkNoMiddleware-4                 632,286    1890 ns/op
BenchmarkSingleMiddleware-4             501,854    2078 ns/op
BenchmarkMultipleMiddleware-4           398,611    3207 ns/op
BenchmarkComplexMiddlewareStack-4       239,594    4949 ns/op
```

### 并发性能

```
BenchmarkConcurrentRequests/SimpleText_Concurrent10-4     12,584    84,742 ns/op
BenchmarkConcurrentRequests/SimpleText_Concurrent50-4      3,891   312,059 ns/op
BenchmarkConcurrentRequests/SimpleText_Concurrent100-4     1,952   606,473 ns/op
```

### 内容处理性能

```
BenchmarkContextJSON-4             2,011,242    716.1 ns/op
BenchmarkContextXML-4                237,769    6124 ns/op
BenchmarkContextString-4           3,662,016    345.1 ns/op
```

### 静态资源性能

```
BenchmarkStaticResource/SmallFile_WithCache-4       3,864    304,397 ns/op    33.64 MB/s
BenchmarkStaticResource/MediumFile_WithCache-4      1,819    697,892 ns/op   146.73 MB/s
BenchmarkStaticResource/LargeFile_WithCache-4         194  5,208,565 ns/op   201.32 MB/s
```

### ORM 性能

```
BenchmarkSelectNoCache-4                   1,050    1,100,821 ns/op
BenchmarkSelectWithCache-4                 3,656      334,870 ns/op
BenchmarkModelInsert-4                       100   18,020,604 ns/op
BenchmarkModelBatchInsert-4                8,881      219,231 ns/op
BenchmarkConcurrentBatchInsert-4          12,687      130,715 ns/op
BenchmarkQueryById-4                       2,232      539,909 ns/op
BenchmarkQueryWithComplexCondition-4       1,570      764,017 ns/op
BenchmarkTransactionBatchOperations-4        369    4,699,809 ns/op
```

ORM 模块的缓存机制能将查询性能提升约 3.3 倍（334,870 vs 1,100,821 ns/op）；批量操作比单条操作效率高约 82 倍（219,231 vs 18,020,604 ns/op），而并发批量插入则进一步将性能提升至非并发版本的约 1.7 倍。

## 项目结构

WebFrame 框架主要包含以下模块：

```
web/            - Web服务器核心模块
  ├── server.go     - HTTP服务器实现
  ├── router.go     - 路由系统
  ├── context.go    - 请求上下文
  ├── middleware.go - 中间件系统
  ├── response.go   - 响应处理
  ├── template.go   - 模板引擎
  └── group.go      - 路由组

orm/            - 对象关系映射模块
  ├── db.go          - 数据库连接管理
  ├── selector.go    - 查询构建器
  ├── inserter.go    - 插入构建器
  ├── updater.go     - 更新构建器
  ├── deleter.go     - 删除构建器
  ├── transaction.go - 事务支持
  └── cache.go       - 查询缓存

cmd/            - 命令行工具
  └── scaffold/     - 项目脚手架工具
```


## 许可证

本项目采用 MIT 许可证