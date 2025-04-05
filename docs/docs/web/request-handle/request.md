# Request Handle

WebFrame 框架提供了完备的请求处理机制，通过 `Context` 上下文对象提供了丰富的 API 用于处理 HTTP 请求和构建响应。

## 上下文

### Context 概述

`Context` 是 WebFrame 框架的核心组件，封装了 HTTP 请求和响应，并提供了丰富的工具方法。每个请求处理函数都会接收到一个 `Context` 实例，通过它可以访问请求数据并构建响应。

```go
func handler(ctx *web.Context) {
    // 使用 Context 处理请求和构建响应
}
```

### Context 结构

`Context` 包含以下主要字段：

```go
type Context struct {
    Req            *http.Request       // HTTP请求对象
    Resp           http.ResponseWriter // HTTP响应写入器
    Param          map[string]string   // 路由参数映射
    RouteURL       string              // 当前路由的URL
    RespStatusCode int                 // 响应状态码
    RespData       []byte              // 响应数据
    unhandled      bool                // 标记是否已处理请求
    tplEngine      Template            // 模板引擎
    UserValues     map[string]any      // 用户自定义值存储
    Context        context.Context     // 标准上下文对象
    aborted        bool                // 标记是否终止处理
    poolManager    pool.PoolManager    // 连接池管理器
}
```

### 流程控制

控制请求处理流程的方法：

#### Abort

终止当前请求的处理流程，后续中间件和处理函数将不会执行：

```go
func (c *Context) Abort()
```

使用示例：

```go
func authMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        token := ctx.GetHeader("Authorization")
        if token == "" {
            ctx.Abort() // 终止处理
            ctx.JSON(401, map[string]string{"error": "unauthorized"})
            return
        }
        next(ctx)
    }
}
```

#### IsAborted

检查请求是否已被终止：

```go
func (c *Context) IsAborted() bool
```

#### Next

在确保请求未被终止的情况下调用下一个处理函数：

```go
func (c *Context) Next(next web.HandlerFunc)
```

使用示例：

```go
func loggingMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        fmt.Println("request start")
        ctx.Next(next) // 调用下一个处理函数
        fmt.Println("request end")
    }
}
```

### 自定义值存储

`Context` 提供了 `UserValues` 字段，可用于在处理链中传递自定义数据：

```go
func userInfoMiddleware(next web.HandlerFunc) web.HandlerFunc {
    return func(ctx *web.Context) {
        // 存储用户信息
        ctx.UserValues["userId"] = 123
        ctx.UserValues["role"] = "admin"
        next(ctx)
    }
}

func handler(ctx *web.Context) {
    // 获取用户信息
    userId := ctx.UserValues["userId"].(int)
    role := ctx.UserValues["role"].(string)
    
    ctx.JSON(200, map[string]interface{}{
        "message": fmt.Sprintf("user %d with role %s", userId, role),
    })
}
```

## 参数获取

WebFrame 提供了丰富的方法来获取不同来源的请求参数，包括查询参数、路径参数和表单参数。所有这些方法都有类型安全的变体，可以自动转换为所需的数据类型。

### 返回值类型

为了支持类型安全的参数获取，框架定义了一系列具有内置错误处理的值类型：

```go
// StringValue 表示带有可选错误的字符串值
type StringValue struct {
    Value string
    Error error
}

// IntValue 表示带有可选错误的整数值
type IntValue struct {
    Value int
    Error error
}

// 其他类型：Int64Value, FloatValue, BoolValue 等
```

这种设计使得参数检查和类型转换更加简洁：

```go
id := ctx.QueryInt("id")
if id.Error != nil {
    ctx.JSON(400, map[string]string{"error": "invalid ID"})
    return
}
// 使用 id.Value (int类型)
```

### 查询参数

从 URL 查询字符串获取参数：

```go
// 获取字符串参数
name := ctx.QueryParam("name")
if name.Error == nil {
    fmt.Println("name:", name.Value)
}

// 获取整数参数
age := ctx.QueryInt("age")
if age.Error == nil {
    fmt.Println("age:", age.Value)
}

// 获取浮点数参数
height := ctx.QueryFloat("height")
if height.Error == nil {
    fmt.Println("height:", height.Value)
}

// 获取布尔参数
active := ctx.QueryBool("active")
if active.Error == nil {
    fmt.Println("activate bool:", active.Value)
}

// 获取所有查询参数
params := ctx.QueryAll()
```

### 路径参数

从路由路径中提取参数（通过路由定义中的`:param`部分）：

```go
// 路由：/users/:id/:role
// 请求：/users/123/admin

// 获取字符串参数
id := ctx.PathParam("id")
role := ctx.PathParam("role")

// 获取转换后的参数
idInt := ctx.PathInt("id")
if idInt.Error == nil {
    fmt.Println("用户ID:", idInt.Value)
}

// 其他类型
userActive := ctx.PathBool("active")
userScore := ctx.PathFloat("score")
```

### 表单参数

处理 POST、PUT 等请求中的表单数据：

```go
// 获取表单字段
name := ctx.FormValue("name")
if name.Error == nil {
    fmt.Println("name:", name.Value)
}

// 获取整数字段
age := ctx.FormInt("age")
if age.Error == nil {
    fmt.Println("age:", age.Value)
}

// 获取布尔字段
active := ctx.FormBool("active")
if active.Error == nil {
    fmt.Println("activate bool:", active.Value)
}

// 获取所有表单值
allFields, err := ctx.FormAll()
if err == nil {
    for key, values := range allFields {
        fmt.Println(key, "=", values)
    }
}
```

### 其他请求信息

获取更多请求相关的信息：

```go
// 获取请求头
authorization := ctx.GetHeader("Authorization")
contentType := ctx.ContentType()

// 检查内容类型
if ctx.IsJSON() {
    fmt.Println("这是 JSON 请求")
}

// 获取客户端信息
clientIP := ctx.ClientIP()
userAgent := ctx.UserAgent()
referer := ctx.Referer()
```

## 请求绑定

WebFrame 支持将请求体内容绑定到 Go 结构体，简化请求数据处理。

### JSON 绑定

将 JSON 请求体绑定到结构体：

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func createUser(ctx *web.Context) {
    var user User
    
    if err := ctx.BindJSON(&user); err != nil {
        ctx.JSON(400, map[string]string{"error": "无效的JSON数据"})
        return
    }
    
    // 使用绑定后的 user 对象
    fmt.Printf("创建用户: %s, 年龄: %d\n", user.Name, user.Age)
    
    ctx.JSON(201, user)
}
```

### XML 绑定

将 XML 请求体绑定到结构体：

```go
type Product struct {
    ID    int    `xml:"id"`
    Name  string `xml:"name"`
    Price float64 `xml:"price"`
}

func createProduct(ctx *web.Context) {
    var product Product
    
    if err := ctx.BindXML(&product); err != nil {
        ctx.JSON(400, map[string]string{"error": "invalid XML data"})
        return
    }
    
    // 使用绑定后的 product 对象
    fmt.Printf("create product: %s, price: %.2f\n", product.Name, product.Price)
    
    ctx.XML(201, product)
}
```

### 原始请求体

如果需要访问原始请求体数据：

```go
func handleRawData(ctx *web.Context) {
    // 读取请求体的原始字节数据
    body, err := ctx.ReadBody()
    if err != nil {
        ctx.JSON(400, map[string]string{"error": "failed to read request body"})
        return
    }
    
    // 处理原始数据
    fmt.Printf("get %d bytes data\n", len(body))
    
    // 返回响应
    ctx.String(200, "get %d bytes data", len(body))
}
```

## 文件处理

### 文件上传

WebFrame 提供了便捷的文件上传处理功能。

#### 基本文件上传

使用 Context 处理单个文件上传：

```go
func uploadHandler(ctx *web.Context) {
    // 获取上传的单个文件
    fileHeader, err := ctx.FormFile("upload_file")
    if err != nil {
        ctx.JSON(400, map[string]string{"error": "failed to get file"})
        return
    }
    
    // 处理文件信息
    fmt.Printf("get file: %s, size: %d 字节\n", fileHeader.Filename, fileHeader.Size)
    
    // 自行实现文件保存逻辑
    // ...
    
    ctx.JSON(200, map[string]string{
        "message": "upload successfully",
        "filename": fileHeader.Filename,
    })
}
```

#### 多文件上传

处理多个文件上传：

```go
func multiUploadHandler(ctx *web.Context) {
    // 获取多个上传文件
    files, err := ctx.FormFiles("upload_files")
    if err != nil {
        ctx.JSON(400, map[string]string{"error": "failed to handle files"})
        return
    }
    
    fileNames := make([]string, 0, len(files))
    
    // 处理每个文件
    for _, file := range files {
        fmt.Printf("get file: %s, size: %d bytes\n", file.Filename, file.Size)
        fileNames = append(fileNames, file.Filename)
        
        // 自行实现文件保存逻辑
        // ...
    }
    
    ctx.JSON(200, map[string]interface{}{
        "message": "upload successfully",
        "files": fileNames,
    })
}
```

#### 高级文件上传处理器

WebFrame 提供了 `FileUploder` 组件，用于更高级的文件上传控制：

```go
func configureFileUpload(s *web.HTTPServer) {
    // 创建文件上传处理器
    uploader := web.NewFileUploader(
        "upload_file",    // 表单字段名
        "./uploads",      // 上传目标目录
        web.WithFileMaxSize(10 << 20),  // 10MB 大小限制
        web.WithAllowedTypes([]string{  // 允许的文件类型
            "image/jpeg",
            "image/png",
            "application/pdf",
        }),
    )
    
    // 注册上传路由
    s.Post("/upload", uploader.HandleUpload())
}
```

`FileUploder` 提供了以下功能：
- 文件大小限制
- 文件类型验证
- 安全的文件名处理
- 目录自动创建
- 自动保存上传文件

### 文件下载

WebFrame 提供了多种方式处理文件下载。

#### 基本文件响应

使用 Context 直接发送文件内容：

```go
func downloadHandler(ctx *web.Context) {
    filePath := "./files/document.pdf"
    
    // 发送文件内容
    ctx.File(filePath)
}
```

#### 文件附件下载

将文件作为附件发送，并设置下载文件名：

```go
func downloadAttachmentHandler(ctx *web.Context) {
    filePath := "./files/document.pdf"
    fileName := "document.pdf" // 指定下载时的文件名
    
    ctx.Attachment(filePath, fileName)
}
```

#### 文件下载处理器

WebFrame 提供了 `FileDownloader` 组件，用于高级文件下载控制：

```go
func configureFileDownload(s *web.HTTPServer) {
    // 创建文件下载处理器
    downloader := web.FileDownloader{
        DestPath: "./files", // 文件存储目录
    }
    
    // 注册下载路由
    s.Get("/download/:file", downloader.HandleDownload())
}
```

`FileDownloader` 会自动：
- 验证请求文件路径的安全性
- 设置合适的内容类型和下载头
- 高效地传输文件内容

## 响应构建

WebFrame 提供了丰富的响应构建方法，支持多种数据格式和用例。

### JSON 响应

```go
func handleJSON(ctx *web.Context) {
    user := map[string]interface{}{
        "id": 123,
        "name": "fyerfyer",
        "roles": []string{"user", "admin"},
    }
    
    ctx.JSON(200, user)
}
```

### XML 响应

```go
func handleXML(ctx *web.Context) {
    product := struct {
        XMLName xml.Name `xml:"product"`
        ID      int      `xml:"id"`
        Name    string   `xml:"name"`
    }{
        ID:   42,
        Name: "WebFrame",
    }
    
    ctx.XML(200, product)
}
```

### 文本响应

```go
func handleText(ctx *web.Context) {
    ctx.String(200, "current time: %s", time.Now().Format(time.RFC3339))
}
```

### HTML 响应

```go
func handleHTML(ctx *web.Context) {
    html := `
        <!DOCTYPE html>
        <html>
        <head><title>WebFrame</title></head>
        <body>
            <h1>welcome to WebFrame</h1>
        </body>
        </html>
    `
    ctx.HTML(200, html)
}
```

### 模板响应

```go
func handleTemplate(ctx *web.Context) {
    data := map[string]interface{}{
        "Title":   "WebFrame",
        "Message": "render page",
        "Items":   []string{"page 1", "page 2", "page 3"},
    }
    
    ctx.Template("home.html", data)
}
```

### 状态码快捷方法

```go
func handleErrors(ctx *web.Context) {
    // 根据条件使用不同的错误响应
    if userNotFound {
        ctx.NotFound("user nonexist")
        return
    }
    
    if unauthorized {
        ctx.Unauthorized("please register first")
        return
    }
    
    if badRequest {
        ctx.BadRequest("无效的请求参数")
        return
    }
    
    if serverError {
        ctx.InternalServerError("处理请求时发生错误")
        return
    }
}
```

### 重定向

```go
func handleRedirect(ctx *web.Context) {
    ctx.Redirect(302, "/new-location")
}
```

## 最佳实践

### 参数验证

始终验证并处理参数错误：

```go
func getUserByID(ctx *web.Context) {
    id := ctx.PathInt("id")
    if id.Error != nil {
        ctx.BadRequest("invalide user ID")
        return
    }
    
    if id.Value <= 0 {
        ctx.BadRequest("user ID must be positive")
        return
    }
    
    // 处理有效ID...
}
```

### 错误处理

使用一致的错误响应格式：

```go
func createResource(ctx *web.Context) {
    var resource Resource
    
    if err := ctx.BindJSON(&resource); err != nil {
        ctx.JSON(400, map[string]string{
            "error": "invalid request data",
            "detail": err.Error(),
        })
        return
    }
    
    // 处理请求...
}
```

### 结构化请求处理

将请求处理分为验证、处理和响应三个阶段：

```go
func updateUser(ctx *web.Context) {
    // 1. 验证阶段
    id := ctx.PathInt("id")
    if id.Error != nil {
        ctx.BadRequest("invalid user ID")
        return
    }
    
    var userData UserUpdateRequest
    if err := ctx.BindJSON(&userData); err != nil {
        ctx.BadRequest("invalid request data")
        return
    }
    
    // 2. 处理阶段
    user, err := userService.Update(id.Value, userData)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            ctx.NotFound("user not found")
        } else {
            ctx.InternalServerError("failed to update user")
            log.Printf("user update error: %v", err)
        }
        return
    }
    
    // 3. 响应阶段
    ctx.JSON(200, user)
}
```

### 文件上传安全性

始终限制文件大小和类型：

```go
uploader := web.NewFileUploader(
    "file",
    "./uploads",
    web.WithFileMaxSize(5 << 20), // 5MB 限制
    web.WithAllowedTypes([]string{
        "image/jpeg",
        "image/png",
        "application/pdf",
    }),
)
```