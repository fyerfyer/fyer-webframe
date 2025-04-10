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

### 路由性能

| 测试场景 | 性能数据 | 说明 |
|---------|---------|------|
| 静态路由 | 797.6 ns/op | 用于处理固定路径的路由 |
| 参数路由 | 1543 ns/op | 处理包含参数的路由如 `/users/:id` |
| 通配符路由 | 876.0 ns/op | 处理包含通配符的路由如 `/api/*` |
| 正则路由 | 1786 ns/op | 处理包含正则表达式的复杂路由 |


### 中间件性能

| 测试场景 | 性能数据 | 说明 |
|---------|---------|------|
| 无中间件 | 1890 ns/op | 基准参考，无中间件处理 |
| 单一中间件 | 2078 ns/op | 使用一个中间件的性能影响 |
| 多个中间件 | 3207 ns/op | 使用多个中间件的堆叠性能 |
| 复杂中间件栈 | 4949 ns/op | 复杂中间件链处理的性能表现 |


### 并发性能

| 测试场景 | 性能数据 | 说明 |
|---------|---------|------|
| 10并发 | 84,742 ns/op | 低并发下的处理能力 |
| 50并发 | 312,059 ns/op | 中等并发下的处理能力 |
| 100并发 | 606,473 ns/op | 高并发下的处理能力 |


### 内容处理性能

| 响应类型 | 性能数据 | 说明 |
|---------|---------|------|
| JSON响应 | 716.1 ns/op | API常用的JSON格式响应 |
| XML响应 | 6124 ns/op | XML格式响应处理 |
| 字符串响应 | 345.1 ns/op | 纯文本响应，性能最佳 |

### 静态资源性能

| 文件大小 | 性能数据 | 吞吐量 | 说明 |
|---------|---------|-------|------|
| 小型文件 | 304,397 ns/op | 33.64 MB/s | 带缓存的小文件处理 |
| 中型文件 | 697,892 ns/op | 146.73 MB/s | 带缓存的中型文件处理 |
| 大型文件 | 5,208,565 ns/op | 201.32 MB/s | 带缓存的大型文件处理 |


### ORM性能

| 测试场景 | 性能数据 | 说明 |
|---------|---------|------|
| 无缓存查询 | 1,100,821 ns/op | 基准查询性能 |
| 缓存查询 | 334,870 ns/op | 启用缓存的查询性能，提升约3.3倍 |
| 单条插入 | 18,020,604 ns/op | 单条记录插入性能 |
| 批量插入 | 219,231 ns/op | 批量记录插入，比单条快约82倍 |
| 并发批量插入 | 130,715 ns/op | 并发批量插入，比普通批量快约1.7倍 |
| ID查询 | 539,909 ns/op | 按主键ID查询的性能 |
| 复杂条件查询 | 764,017 ns/op | 复杂查询条件的性能表现 |
| 事务批量操作 | 4,699,809 ns/op | 事务内的批量操作性能 |

ORM 模块的缓存机制能将查询性能提升约 3.3 倍（334,870 vs 1,100,821 ns/op）；批量操作比单条操作效率高约 82 倍（219,231 vs 18,020,604 ns/op），而并发批量插入则进一步将性能提升至非并发版本的约 1.7 倍。

> 注：以上性能测试在Intel Core i5-4310U CPU @ 2.00GHz, Windows环境下进行，实际性能可能因硬件配置、网络环境和系统负载而有所不同。

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