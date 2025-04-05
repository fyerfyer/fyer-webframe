# 模板引擎

WebFrame 框架集成了一个强大而灵活的模板引擎，基于 Go 的标准模板库构建，同时提供了许多增强功能，如自动重载、布局模板支持和自定义函数。本文档详细介绍了模板引擎的使用方法，从基础用法到高级特性。

## 基础用法

### 初始化模板引擎

```go
// 创建基本模板引擎
tpl := web.NewGoTemplate()

// 使用选项模式配置模板引擎
tpl := web.NewGoTemplate(
    web.WithPattern("./templates/*.html"), // 从模式加载模板
    web.WithAutoReload(true),             // 开启自动重载
)
```

### 配置模板引擎选项

WebFrame 使用选项模式进行模板引擎的配置：

```go
// 从特定模式加载模板
tpl := web.NewGoTemplate(web.WithPattern("./views/*.html"))

// 从指定文件列表加载模板
tpl := web.NewGoTemplate(web.WithFiles(
    "./views/layout.html",
    "./views/home.html",
    "./views/user.html",
))

// 启用自动重载（开发环境推荐）
tpl := web.NewGoTemplate(web.WithAutoReload(true))
```

### 集成到 HTTP 服务器

将模板引擎集成到 WebFrame HTTP 服务器:

```go
// 方法 1: 在创建服务器时设置
tpl := web.NewGoTemplate(web.WithPattern("./templates/*.html"))
server := web.NewHTTPServer(web.WithTemplate(tpl))

// 方法 2: 使用服务器接口方法
tpl := web.NewGoTemplate(web.WithPattern("./templates/*.html"))
server := web.NewHTTPServer()
server.UseTemplate(tpl)
```

### 渲染模板

在处理函数中使用模板引擎渲染页面:

```go
// 注册一个使用模板的路由处理器
server.Get("/", func(ctx *web.Context) {
    // 准备模板数据
    data := map[string]interface{}{
        "Title":   "Welcome to WebFrame",
        "Message": "Hello, World!",
        "User": map[string]string{
            "Name": "John Doe",
            "Role": "Admin",
        },
        "Items": []string{"Item 1", "Item 2", "Item 3"},
    }
    
    // 渲染模板
    err := ctx.Template("home.html", data)
    if err != nil {
        ctx.InternalServerError("Failed to render template")
        return
    }
})
```

### 管理模板文件

在运行时加载和重新加载模板:

```go
// 从 glob 模式加载模板
if err := tpl.LoadFromGlob("./views/*.html"); err != nil {
    log.Printf("Failed to load templates: %v", err)
    return
}

// 从特定文件列表加载模板
if err := tpl.LoadFromFiles("./views/layout.html", "./views/page.html"); err != nil {
    log.Printf("Failed to load templates: %v", err)
    return
}

// 手动重新加载模板（在使用自动重载时通常不需要）
if err := tpl.Reload(); err != nil {
    log.Printf("Failed to reload templates: %v", err)
    return
}
```

## 自定义函数

WebFrame 支持在模板中使用自定义函数，极大地增强了模板的灵活性和功能。

### 注册自定义函数

使用 `WithFuncMap` 选项添加自定义模板函数：

```go
// 创建包含自定义函数的模板引擎
tpl := web.NewGoTemplate(
    web.WithPattern("./templates/*.html"),
    web.WithFuncMap(template.FuncMap{
        // 格式化日期时间
        "formatDate": func(t time.Time, layout string) string {
            return t.Format(layout)
        },
        
        // 截取字符串
        "truncate": func(s string, length int) string {
            if len(s) <= length {
                return s
            }
            return s[:length] + "..."
        },
        
        // 生成 URL
        "url": func(path string, params ...string) string {
            // 基本路径
            result := path
            
            // 添加查询参数
            if len(params) > 0 && len(params)%2 == 0 {
                result += "?"
                for i := 0; i < len(params); i += 2 {
                    if i > 0 {
                        result += "&"
                    }
                    result += params[i] + "=" + url.QueryEscape(params[i+1])
                }
            }
            
            return result
        },
        
        // 条件判断
        "ifThen": func(condition bool, then, otherwise interface{}) interface{} {
            if condition {
                return then
            }
            return otherwise
        },
    }),
)
```

### 在模板中使用自定义函数

模板中使用自定义函数的例子：

```html
<!-- 在模板中使用自定义函数 -->
<div class="article">
    <h2>{{ .Title }}</h2>
    <p class="timestamp">Published: {{ formatDate .PublishedAt "Jan 2, 2006" }}</p>
    
    <div class="summary">
        {{ truncate .Content 200 }}
    </div>
    
    <a href="{{ url "/article" "id" .ID }}">Read more</a>
    
    <div class="status">
        Status: {{ ifThen .IsPublished "Published" "Draft" }}
    </div>
</div>
```

### 常用自定义函数示例

以下是一些在 Web 应用中常用的模板函数：

```go
funcMap := template.FuncMap{
    // 数值函数
    "add": func(a, b int) int {
        return a + b
    },
    "subtract": func(a, b int) int {
        return a - b
    },
    "multiply": func(a, b int) int {
        return a * b
    },
    "divide": func(a, b int) float64 {
        if b == 0 {
            return 0
        }
        return float64(a) / float64(b)
    },
    
    // 字符串函数
    "lower": strings.ToLower,
    "upper": strings.ToUpper,
    "title": strings.Title,
    
    // 切片函数
    "first": func(items []interface{}) interface{} {
        if len(items) == 0 {
            return nil
        }
        return items[0]
    },
    "last": func(items []interface{}) interface{} {
        if len(items) == 0 {
            return nil
        }
        return items[len(items)-1]
    },
    "join": strings.Join,
    
    // HTML 安全
    "safeHTML": func(s string) template.HTML {
        return template.HTML(s)
    },
    
    // JSON 格式化
    "json": func(v interface{}) string {
        b, err := json.Marshal(v)
        if err != nil {
            return ""
        }
        return string(b)
    },
}
```

## 布局模板

WebFrame 支持布局模板（也称为母版页）。

### 创建布局模板

布局模板定义了页面的整体结构，包含子模板将填充的内容区域：

```html
<!-- layout.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - WebFrame</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    {{block "styles" .}}{{end}}
</head>
<body>
    <header>
        <h1>WebFrame Application</h1>
        <nav>
            <a href="/">Home</a>
            <a href="/about">About</a>
            <a href="/contact">Contact</a>
        </nav>
    </header>
    
    <main>
        {{block "content" .}}
            <!-- 默认内容将被覆盖 -->
            <p>Default content</p>
        {{end}}
    </main>
    
    <footer>
        <p>&copy; {{.CurrentYear}} WebFrame Example</p>
    </footer>
    
    <script src="/static/js/main.js"></script>
    {{block "scripts" .}}{{end}}
</body>
</html>
```

### 创建内容模板

内容模板使用 `{{define}}` 和 `{{template}}` 块定义特定页面的内容:

```html
<!-- home.html -->
{{define "content"}}
<div class="home-page">
    <h2>{{.Title}}</h2>
    <p>{{.Message}}</p>
    
    {{if .Items}}
        <ul>
            {{range .Items}}
                <li>{{.}}</li>
            {{end}}
        </ul>
    {{end}}
</div>
{{end}}

{{define "styles"}}
<style>
    .home-page {
        background-color: #f9f9f9;
        padding: 20px;
    }
</style>
{{end}}
```

### 使用模板组合

在 WebFrame 中加载和使用模板组合:

```go
func main() {
    // 创建并配置模板引擎
    tpl := web.NewGoTemplate(
        // 注意顺序：先加载布局模板，再加载内容模板
        web.WithFiles("./views/layout.html", "./views/home.html", "./views/about.html"),
    )
    
    server := web.NewHTTPServer(web.WithTemplate(tpl))
    
    // 使用模板作为页面
    server.Get("/", func(ctx *web.Context) {
        data := map[string]interface{}{
            "Title":       "Home Page",
            "Message":     "Welcome to our website!",
            "Items":       []string{"Item 1", "Item 2", "Item 3"},
            "CurrentYear": time.Now().Format("2006"),
        }
        
        // 渲染布局模板，它将包含 home.html 中定义的内容
        ctx.Template("layout.html", data)
    })
    
    server.Get("/about", func(ctx *web.Context) {
        data := map[string]interface{}{
            "Title":       "About Us",
            "Message":     "Learn more about our company",
            "CurrentYear": time.Now().Format("2006"),
        }
        
        // 与首页使用相同的布局，但渲染不同的内容模板
        ctx.Template("layout.html", data)
    })
    
    server.Start(":8080")
}
```

### 嵌套模板和部分视图

创建可重用的部分视图，进一步提高代码复用:

```html
<!-- partials/header.html -->
{{define "header"}}
<header>
    <h1>{{.SiteName}}</h1>
    <nav>
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/contact">Contact</a>
        {{if .User}}
            <a href="/profile">Welcome, {{.User.Name}}</a>
            <a href="/logout">Logout</a>
        {{else}}
            <a href="/login">Login</a>
            <a href="/register">Register</a>
        {{end}}
    </nav>
</header>
{{end}}
```

在布局模板中使用部分视图:

```html
<!-- layout.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - {{.SiteName}}</title>
    <!-- ... -->
</head>
<body>
    {{template "header" .}}
    
    <main>
        {{block "content" .}}{{end}}
    </main>
    
    {{template "footer" .}}
    <!-- ... -->
</body>
</html>
```

### 高级布局技巧

#### 多级布局

创建多级布局结构，适用于复杂网站:

```html
<!-- base.html - 最基础的布局 -->
<!DOCTYPE html>
<html>
<!-- ... -->
<body>
    {{template "header" .}}
    <div class="container">
        {{block "layout" .}}{{end}}
    </div>
    {{template "footer" .}}
</body>
</html>

<!-- dashboard-layout.html - 仪表板特有布局 -->
{{define "layout"}}
<div class="dashboard">
    <aside class="sidebar">
        {{template "sidebar" .}}
    </aside>
    <main class="content">
        {{block "dashboard-content" .}}{{end}}
    </main>
</div>
{{end}}

<!-- dashboard-users.html - 特定页面内容 -->
{{define "dashboard-content"}}
<h2>Users Management</h2>
<table class="users-table">
    <!-- Users table content -->
</table>
{{end}}
```

#### 数据组合

使用嵌套模板时合并数据:

```go
server.Get("/dashboard/users", func(ctx *web.Context) {
    // 基本页面数据
    baseData := map[string]interface{}{
        "Title":       "User Management",
        "SiteName":    "Admin Portal",
        "CurrentYear": time.Now().Format("2006"),
        "User": map[string]string{
            "Name": "Admin",
            "Role": "Administrator",
        },
    }
    
    // 页面特定数据
    usersData := map[string]interface{}{
        "Users": []map[string]interface{}{
            {"ID": 1, "Name": "John", "Email": "john@example.com"},
            {"ID": 2, "Name": "Jane", "Email": "jane@example.com"},
            {"ID": 3, "Name": "Bob", "Email": "bob@example.com"},
        },
        "TotalUsers": 3,
    }
    
    // 合并数据
    data := baseData
    for k, v := range usersData {
        data[k] = v
    }
    
    // 渲染模板链
    ctx.Template("base.html", data)
})
```