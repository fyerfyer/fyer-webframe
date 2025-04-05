# Router

WebFrame 支持多种路由模式、路由分组、参数提取和静态资源服务。

## 路由注册

### 基础路由注册

WebFrame 支持所有标准的 HTTP 方法，通过简单的 API 进行路由注册：

```go
func main() {
    server := web.NewHTTPServer()
    
    // 注册 GET 路由
    server.Get("/hello", func(ctx *web.Context) {
        ctx.String(200, "Hello World!")
    })
    
    // 注册 POST 路由
    server.Post("/users", func(ctx *web.Context) {
        // 处理创建用户
        ctx.JSON(201, map[string]string{"id": "123", "name": "new user"})
    })
    
    // 注册 PUT 路由
    server.Put("/users/:id", func(ctx *web.Context) {
        id := ctx.PathParam("id").Value
        ctx.String(200, "update user: "+id)
    })
    
    // 注册 DELETE 路由
    server.Delete("/users/:id", func(ctx *web.Context) {
        id := ctx.PathParam("id").Value
        ctx.String(200, "delete user: "+id)
    })
    
    // 注册 PATCH 路由
    server.Patch("/users/:id/status", func(ctx *web.Context) {
        id := ctx.PathParam("id").Value
        ctx.String(200, "update user status: "+id)
    })
    
    // 注册 OPTIONS 路由
    server.Options("/users", func(ctx *web.Context) {
        ctx.Resp.Header().Set("Allow", "GET, POST, PUT, DELETE")
        ctx.Status(204)
    })
    
    server.Start(":8080")
}
```

### 链式 API

WebFrame 支持链式 API 风格，可以在路由注册后直接添加中间件：

```go
server.Get("/admin/dashboard", func(ctx *web.Context) {
    ctx.String(200, "admin dashbord")
}).Middleware(
    // 添加认证中间件
    func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            // 验证权限
            if !isAdmin(ctx) {
                ctx.String(403, "access denied")
                return
            }
            next(ctx)
        }
    },
    // 添加日志中间件
    func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            fmt.Println("access admin dashboard")
            next(ctx)
        }
    },
)
```

## 路由组

路由组允许您将相关路由组织在一起，共享公共前缀和中间件。

### 创建路由组

```go
func main() {
    server := web.NewHTTPServer()
    
    // 创建 API 路由组
    api := server.Group("/api")
    
    // 注册 API 路由
    api.Get("/users", listUsers)
    api.Post("/users", createUser)
    
    // 创建 v1 版本 API 子组
    v1 := api.Group("/v1")
    v1.Get("/products", listProductsV1)
    
    // 创建 v2 版本 API 子组
    v2 := api.Group("/v2")
    v2.Get("/products", listProductsV2)
    
    server.Start(":8080")
}
```

### 组级中间件

可以为整个路由组添加中间件，这些中间件将应用于该组中的所有路由：

```go
// 创建路由组并添加中间件
usersGroup := server.Group("/users").Use(
    authMiddleware,
    loggingMiddleware,
)

// 组中的所有路由都会应用上面的中间件
usersGroup.Get("", listUsers)
usersGroup.Get("/:id", getUserById)
usersGroup.Post("", createUser)
```

### 嵌套路由组

路由组可以无限嵌套，每个子组继承父组的路径前缀和中间件：

```go
// 主 API 组
api := server.Group("/api")

// 认证 API 子组
auth := api.Group("/auth")
auth.Post("/login", handleLogin)
auth.Post("/register", handleRegister)

// 用户 API 子组
users := api.Group("/users").Use(authRequired)
users.Get("", listUsers)

// 用户文档子组
userDocs := users.Group("/:id/documents")
userDocs.Get("", listUserDocuments)
userDocs.Post("", uploadUserDocument)
```

## 路由参数

WebFrame 支持多种类型的路由参数，能够满足各种复杂的 URL 匹配需求。

### 参数路由

使用 `:param` 语法定义路径参数：

```go
server.Get("/users/:id", func(ctx *web.Context) {
    id := ctx.PathParam("id").Value
    ctx.String(200, "用户 ID: "+id)
})

server.Get("/blogs/:year/:month/:day/:slug", func(ctx *web.Context) {
    year := ctx.PathParam("year").Value
    month := ctx.PathParam("month").Value
    day := ctx.PathParam("day").Value
    slug := ctx.PathParam("slug").Value
    
    ctx.JSON(200, map[string]string{
        "year": year,
        "month": month,
        "day": day,
        "slug": slug,
    })
})
```

参数值可以通过 `Context` 对象的方法获取并转换为所需类型：

```go
server.Get("/products/:id/reviews/:score", func(ctx *web.Context) {
    // 获取字符串参数
    idStr := ctx.PathParam("id").Value
    
    // 获取并转换为整数
    id := ctx.PathInt("id").Value
    
    // 获取并转换为浮点数
    score := ctx.PathFloat("score").Value
    
    ctx.JSON(200, map[string]interface{}{
        "product_id": id,
        "score": score,
    })
})
```

### 正则路由参数

可以使用正则表达式限制参数格式：

```go
// 限制 ID 只能是数字
server.Get("/users/:id([0-9]+)", func(ctx *web.Context) {
    id := ctx.PathParam("id").Value
    ctx.String(200, "用户 ID (数字): "+id)
})

// 限制用户名只能是字母
server.Get("/users/:username([a-zA-Z]+)", func(ctx *web.Context) {
    username := ctx.PathParam("username").Value
    ctx.String(200, "用户名 (字母): "+username)
})

// 更复杂的正则表达式
server.Get("/articles/:slug([a-z0-9-]+)", func(ctx *web.Context) {
    slug := ctx.PathParam("slug").Value
    ctx.String(200, "文章 Slug: "+slug)
})
```

### 通配符路由

使用 `*` 匹配任意路径段：

```go
// 匹配 /files/ 后的任何路径
server.Get("/files/*", func(ctx *web.Context) {
    path := ctx.PathParam("file").Value
    ctx.String(200, "请求的文件路径: "+path)
})

// 处理所有未找到的路由
server.Get("/*", func(ctx *web.Context) {
    ctx.String(404, "未找到页面")
})
```

### 路由匹配优先级

WebFrame 路由匹配遵循以下优先级规则：

1. **静态路由**：完全匹配的路径，如 `/users/profile`
2. **正则路由**：包含正则表达式的参数路径，如 `/users/:id([0-9]+)`
3. **参数路由**：包含参数的路径，如 `/users/:id`
4. **通配符路由**：包含通配符的路径，如 `/users/*`

例如，对于请求 `/users/123`，匹配顺序为：

```go
server.Get("/users/123", func(ctx *web.Context) {
    // 1. 首先匹配这个静态路由
})

server.Get("/users/:id([0-9]+)", func(ctx *web.Context) {
    // 2. 其次匹配这个正则路由
})

server.Get("/users/:id", func(ctx *web.Context) {
    // 3. 然后匹配这个参数路由
})

server.Get("/users/*", func(ctx *web.Context) {
    // 4. 最后匹配这个通配符路由
})
```

## 静态资源路由

WebFrame 提供了内置支持，用于服务静态文件，如 CSS、JavaScript、图片等。

### 基本用法

```go
func main() {
    server := web.NewHTTPServer()
    
    // 创建静态资源处理器
    staticResource := web.NewStaticResource("./static")
    
    // 注册静态资源路由
    server.Use("GET", "/static/*", staticResource.Handle())
    
    // 启动服务器
    server.Start(":8080")
}
```

### 高级配置

静态资源处理器支持多种配置选项：

```go
func main() {
    server := web.NewHTTPServer()
    
    // 创建带配置的静态资源处理器
    staticResource := web.NewStaticResource(
        "./public",
        web.WithPathPrefix("/assets/"),
        web.WithMaxSize(10 << 20), // 10MB 缓存限制
        web.WithCache(time.Hour, 10*time.Minute), // 缓存配置
        web.WithExtContentTypes(map[string]string{
            ".css":  "text/css; charset=utf-8",
            ".js":   "application/javascript",
            ".png":  "image/png",
            ".jpg":  "image/jpeg",
            ".jpeg": "image/jpeg",
            ".gif":  "image/gif",
            ".svg":  "image/svg+xml",
            ".woff": "font/woff",
            ".woff2": "font/woff2",
        }),
    )
    
    // 注册静态资源路由
    server.Use("GET", "/assets/*", staticResource.Handle())
    
    server.Start(":8080")
}
```

### 文件上传和下载

WebFrame 也提供了文件上传和下载的处理器：

```go
func main() {
    server := web.NewHTTPServer()
    
    // 文件上传处理
    uploader := web.NewFileUploader(
        "upload_file", // 表单字段名
        "./uploads",   // 上传目标目录
        web.WithFileMaxSize(50 << 20), // 限制 50MB
        web.WithAllowedTypes([]string{ // 限制文件类型
            "image/jpeg",
            "image/png",
            "application/pdf",
        }),
    )
    server.Post("/upload", uploader.HandleUpload())
    
    // 文件下载处理
    downloader := web.FileDownloader{
        DestPath: "./downloads",
    }
    server.Get("/download/:file", downloader.HandleDownload())
    
    server.Start(":8080")
}
```

## 综合示例

下面是一个综合使用路由系统各种功能的完整示例：

```go
package main

import (
    "fmt"
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"
    "net/http"
    "time"
)

func main() {
    // 创建服务器
    server := web.NewHTTPServer()
    
    // 添加全局中间件
    server.Use("*", "*", recovery.Recovery())
    server.Use("*", "*", accesslog.NewMiddlewareBuilder().Build())
    
    // 静态资源配置
    staticFiles := web.NewStaticResource(
        "./public",
        web.WithPathPrefix("/static/"),
        web.WithCache(time.Hour, 10*time.Minute),
    )
    server.Use("GET", "/static/*", staticFiles.Handle())
    
    // 基本路由
    server.Get("/", func(ctx *web.Context) {
        ctx.HTML(200, "<h1>欢迎使用 WebFrame</h1>")
    })
    
    // API 路由组
    api := server.Group("/api")
    
    // v1 API 版本
    v1 := api.Group("/v1")
    
    // 认证子组
    auth := v1.Group("/auth")
    auth.Post("/login", handleLogin)
    auth.Post("/register", handleRegister)
    
    // 用户子组 (需要认证)
    users := v1.Group("/users").Use(authMiddleware)
    users.Get("", listUsers)
    users.Get("/:id([0-9]+)", getUserById)
    users.Put("/:id([0-9]+)", updateUser)
    users.Delete("/:id([0-9]+)", deleteUser)
    
    // 用户文档子组
    docs := users.Group("/:user_id([0-9]+)/documents")
    docs.Get("", listUserDocuments)
    docs.Get("/:doc_id", getUserDocument)
    docs.Post("", uploadUserDocument)
    
    fmt.Println("服务器启动在 :8080")
    server.Start(":8080")
}

// 处理函数
func handleLogin(ctx *web.Context) {
    // 登录逻辑
}

func handleRegister(ctx *web.Context) {
    // 注册逻辑
}

func listUsers(ctx *web.Context) {
    // 列出用户
    ctx.JSON(200, []map[string]interface{}{
        {"id": 1, "name": "用户1"},
        {"id": 2, "name": "用户2"},
    })
}

func getUserById(ctx *web.Context) {
    id := ctx.PathInt("id").Value
    ctx.JSON(200, map[string]interface{}{
        "id": id,
        "name": fmt.Sprintf("用户%d", id),
    })
}

func updateUser(ctx *web.Context) {
    // 更新用户
}

func deleteUser(ctx *web.Context) {
    // 删除用户
}

func listUserDocuments(ctx *web.Context) {
    userId := ctx.PathInt("user_id").Value
    ctx.JSON(200, map[string]interface{}{
        "user_id": userId,
        "documents": []map[string]interface{}{
            {"id": 1, "name": "文档1"},
            {"id": 2, "name": "文档2"},
        },
    })
}

func getUserDocument(ctx *web.Context) {
    userId := ctx.PathInt("user_id").Value
    docId := ctx.PathParam("doc_id").Value
    ctx.JSON(200, map[string]interface{}{
        "user_id": userId,
        "doc_id": docId,
        "name": "用户文档",
    })
}

func uploadUserDocument(ctx *web.Context) {
    // 上传文档逻辑
}

// 中间件
func authMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        token := ctx.GetHeader("Authorization")
        if token == "" {
            ctx.JSON(http.StatusUnauthorized, map[string]string{
                "error": "未授权访问",
            })
            return
        }
        // 在实际应用中验证令牌
        next(ctx)
    }
}
```