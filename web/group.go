package web

import "path"

// RouteGroup 路由组接口
type RouteGroup interface {
    // Get ...路由注册方法
    Get(path string, handler HandlerFunc) RouteRegister
    Post(path string, handler HandlerFunc) RouteRegister
    Put(path string, handler HandlerFunc) RouteRegister
    Delete(path string, handler HandlerFunc) RouteRegister
    Patch(path string, handler HandlerFunc) RouteRegister
    Options(path string, handler HandlerFunc) RouteRegister
    
    // Group 嵌套组
    Group(prefix string) RouteGroup
    
    // Use 组级中间件
    Use(middleware ...Middleware) RouteGroup
}

// routeGroup 实现 RouteGroup 接口，代表一个路由分组
type routeGroup struct {
    server   *HTTPServer // 指向服务器实例的引用
    basePath string      // 路由组前缀
}

// newRouteGroup 创建一个新的路由组
func newRouteGroup(server *HTTPServer, prefix string) *routeGroup {
    // 确保前缀始终以 / 开头
    if len(prefix) > 0 && prefix[0] != '/' {
        prefix = "/" + prefix
    }

    // 如果前缀以 / 结尾，去掉末尾的 /
    if len(prefix) > 1 && prefix[len(prefix)-1] == '/' {
        prefix = prefix[:len(prefix)-1]
    }

    return &routeGroup{
        server:   server,
        basePath: prefix,
    }
}

// normalizePath 规范化路径，确保路径格式正确
func (g *routeGroup) normalizePath(relativePath string) string {
    if len(relativePath) == 0 {
        return g.basePath
    }

    // 确保相对路径以 / 开头
    if relativePath[0] != '/' {
        relativePath = "/" + relativePath
    }

    // 使用 path.Join 来正确连接路径，并确保结果总是以 / 开头
    result := path.Join(g.basePath, relativePath)
    if result[0] != '/' {
        result = "/" + result
    }

    // 保留原始路径中的尾部斜杠
    if len(relativePath) > 1 && relativePath[len(relativePath)-1] == '/' && result[len(result)-1] != '/' {
        result = result + "/"
    }

    return result
}

// Get 注册 GET 路由方法
func (g *routeGroup) Get(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Get(fullPath, handler)
    return newRouteRegister(g.server, "GET", fullPath)
}

// Post 注册 POST 路由方法
func (g *routeGroup) Post(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Post(fullPath, handler)
    return newRouteRegister(g.server, "POST", fullPath)
}

// Put 注册 PUT 路由方法
func (g *routeGroup) Put(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Put(fullPath, handler)
    return newRouteRegister(g.server, "PUT", fullPath)
}

// Delete 注册 DELETE 路由方法
func (g *routeGroup) Delete(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Delete(fullPath, handler)
    return newRouteRegister(g.server, "DELETE", fullPath)
}

// Patch 注册 PATCH 路由方法
func (g *routeGroup) Patch(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Patch(fullPath, handler)
    return newRouteRegister(g.server, "PATCH", fullPath)
}

// Options 注册 OPTIONS 路由方法
func (g *routeGroup) Options(relativePath string, handler HandlerFunc) RouteRegister {
    fullPath := g.normalizePath(relativePath)
    g.server.Router.Options(fullPath, handler)
    return newRouteRegister(g.server, "OPTIONS", fullPath)
}

// Group 创建嵌套路由组
func (g *routeGroup) Group(relativePath string) RouteGroup {
    return newRouteGroup(g.server, g.normalizePath(relativePath))
}

// Use 为路由组添加中间件
func (g *routeGroup) Use(middleware ...Middleware) RouteGroup {
    // 将中间件应用到该组的所有路由
    for _, m := range middleware {
        // 使用通配符将中间件应用到当前组及其所有子路由
        g.server.Use("GET", g.basePath+"/*", m)
        g.server.Use("POST", g.basePath+"/*", m)
        g.server.Use("PUT", g.basePath+"/*", m)
        g.server.Use("DELETE", g.basePath+"/*", m)
        g.server.Use("PATCH", g.basePath+"/*", m)
        g.server.Use("OPTIONS", g.basePath+"/*", m)
    }
    return g
}