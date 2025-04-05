# Getting Started

下面通过一些简单的实例了解如何使用fyer-webframe创建、启动和调用一个Web服务。

## 安装WebFrame

首先，使用Go模块安装WebFrame框架及其脚手架工具：

```bash
go get github.com/fyerfyer/fyer-webframe
```

安装脚手架命令行工具：

```bash
go install github.com/fyerfyer/fyer-webframe/cmd/scaffold@latest
```

## 使用脚手架创建项目

安装完成后，您可以使用scaffold命令行工具创建一个新项目：

```bash
scaffold -name myproject
```

这个命令会在当前目录下创建一个名为`myproject`的新项目文件夹。

您也可以指定更多选项来自定义项目：

```bash
Options:
  -module string
        Go module path (default: github.com/{project-name})
  -name string
        Project name (required)
  -output string
        Output directory (default: ./{project-name})
  -run
        Run the project after creation

Examples:
  scaffold -name myproject
  scaffold -name myproject -module example.com/myproject
  scaffold -name myproject -output ./projects/myproject
  scaffold -name myproject -run
```

## 项目结构

成功创建项目后，您将看到以下项目结构：

```
myproject/
.
├── ./config
│   └── ./config/config.go
├── ./controllers
│   └── ./controllers/home.go
├── ./go.mod
├── ./go.sum
├── ./main.go
├── ./middlewares
├── ./models
│   └── ./models/user.go
├── ./public
│   ├── ./public/css
│   ├── ./public/images
│   └── ./public/js
└── ./views
    ├── ./views/home.html
    └── ./views/layout.html
```

## 启动项目

进入项目目录，运行项目：

```bash
cd myproject
go run .
```

访问`http://localhost:8080`，即可查看默认的欢迎页面。

# 第一个接口

本指南将带您快速构建一个简单的REST API接口，展示WebFrame框架的基本用法。

## 创建处理器

首先，在`internal/handler`目录下创建一个新的处理器文件`user_handler.go`：

```go
package handler

import (
    "github.com/fyerfyer/fyer-webframe/web"
)

// User 表示用户数据结构
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// UserHandler 处理用户相关请求
type UserHandler struct {
    // 实际项目中可能会注入服务层依赖
    // userService service.UserService
    users map[int]User // 简化示例，使用内存存储
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler() *UserHandler {
    // 初始化一些示例用户数据
    return &UserHandler{
        users: map[int]User{
            1: {ID: 1, Name: "Alice", Age: 28},
            2: {ID: 2, Name: "Bob", Age: 32},
            3: {ID: 3, Name: "Charlie", Age: 25},
        },
    }
}

// GetUsers 获取所有用户
func (h *UserHandler) GetUsers(ctx *web.Context) {
    userList := make([]User, 0, len(h.users))
    for _, user := range h.users {
        userList = append(userList, user)
    }
    ctx.JSON(200, userList)
}

// GetUserByID 根据ID获取单个用户
func (h *UserHandler) GetUserByID(ctx *web.Context) {
    id := ctx.PathParamInt("id", 0)
    if id == 0 {
        ctx.JSON(400, map[string]string{"error": "无效的用户ID"})
        return
    }
    
    user, ok := h.users[id]
    if !ok {
        ctx.JSON(404, map[string]string{"error": "用户不存在"})
        return
    }
    
    ctx.JSON(200, user)
}

// CreateUser 创建新用户
func (h *UserHandler) CreateUser(ctx *web.Context) {
    var user User
    if err := ctx.BindJSON(&user); err != nil {
        ctx.JSON(400, map[string]string{"error": "无效的请求数据"})
        return
    }
    
    // 简化示例：自动生成ID
    maxID := 0
    for id := range h.users {
        if id > maxID {
            maxID = id
        }
    }
    user.ID = maxID + 1
    
    // 保存用户
    h.users[user.ID] = user
    
    ctx.JSON(201, user)
}
```

## 注册路由

在主程序中注册刚刚创建的处理器和路由。修改`main.go`文件：

```go
package main

import (
    "log"
    
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"
    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"
    
    "myproject/internal/handler"
)

func main() {
    // 创建HTTP服务器
    server := web.NewHTTPServer()
    
    // 注册全局中间件
    server.Use("*", "*", recovery.Recovery())
    server.Use("*", "*", accesslog.NewMiddlewareBuilder().Build())
    
    // 创建用户处理器
    userHandler := handler.NewUserHandler()
    
    // 创建API路由组
    apiGroup := server.Group("/api")
    
    // 注册用户相关路由
    apiGroup.Get("/users", userHandler.GetUsers)
    apiGroup.Get("/users/:id([0-9]+)", userHandler.GetUserByID)
    apiGroup.Post("/users", userHandler.CreateUser)
    
    // 启动服务器
    log.Println("Server starting on :8080")
    err := server.Start(":8080")
    if err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}
```

## 测试API

启动服务器：

```bash
go run .
```

现在可以使用curl或其他HTTP客户端工具测试您的API：

## 添加中间件

让我们为用户API添加一个简单的日志中间件。创建`internal/middleware/logger.go`文件：

```go
package middleware

import (
    "log"
    "time"
    
    "github.com/fyerfyer/fyer-webframe/web"
)

// Logger 创建一个记录请求时间的中间件
func Logger() web.Middleware {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            start := time.Now()
            
            // 调用下一个处理器
            next(ctx)
            
            // 计算处理时间
            duration := time.Since(start)
            
            // 记录请求信息和处理时间
            log.Printf("[%s] %s %s - %v", ctx.Method, ctx.Path, ctx.ClientIP(), duration)
        }
    }
}
```

将中间件添加到用户路由：

```go
// ...

// 注册用户相关路由和中间件
userGroup := apiGroup.Group("/users")
userGroup.Use(middleware.Logger()) // 添加自定义中间件

userGroup.Get("", userHandler.GetUsers)
userGroup.Get("/:id([0-9]+)", userHandler.GetUserByID)
userGroup.Post("", userHandler.CreateUser)

// ...
```

## 使用静态资源

为应用程序添加静态资源支持。首先，在`static`目录下创建一个简单的HTML文件`index.html`：

```html
<!DOCTYPE html>
<html>
<head>
    <title>WebFrame示例应用</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        .api-list {
            background-color: #f5f5f5;
            padding: 15px;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <h1>WebFrame示例应用</h1>
    <p>这是一个使用WebFrame构建的示例应用程序。</p>
    
    <h2>可用API端点：</h2>
    <div class="api-list">
        <ul>
            <li>GET /api/users - 获取所有用户</li>
            <li>GET /api/users/:id - 获取指定ID的用户</li>
            <li>POST /api/users - 创建新用户</li>
        </ul>
    </div>
    
    <script>
        console.log('WebFrame示例应用已加载');
    </script>
</body>
</html>
```

然后，修改`main.go`来提供静态资源：

```go
// ...

// 静态资源处理
staticResource := web.NewStaticResource(
    web.WithPathPrefix("/static/"),
    web.WithDirPath("./static"),
)
server.Use("GET", "/static/*", staticResource.Handle)

// 添加根路由重定向到静态页面
server.Get("/", func(ctx *web.Context) {
    ctx.Redirect("/static/index.html")
})

// ...
```
