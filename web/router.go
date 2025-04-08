package web

import (
	"github.com/fyerfyer/fyer-webframe/web/router"
	"strings"
)

// Router 路由器结构体
type Router struct {
	routerTrees map[string]*node     // 用于向后兼容的路由树结构
	middlewares map[string][]MiddlewareWithPath // 使用http方法作为键值对
	orderCounter int                 // 用于记录中间件注册顺序
	radixRouter  *router.Router      // 使用RadixTree实现的新路由器
}

// node 节点结构，用于向后兼容
type node struct {
	path                string
	children            map[string]*node
	parent              *node
	handler             HandlerFunc
	hasStarParam        bool
	isParam             bool
	hasParamChild       bool
	hasStarChild        bool
	Param               map[string]string
	isRegex             bool
	regexPattern        interface{} // 改为interface{}避免循环导入
	staticMiddlewares   []MiddlewareWithPath
	regexMiddlewares    []MiddlewareWithPath
	paramMiddlewares    []MiddlewareWithPath
	wildcardMiddlewares []MiddlewareWithPath
}

// NewRouter 创建一个新的路由器
func NewRouter() *Router {
	return &Router{
		routerTrees: make(map[string]*node),
		middlewares: make(map[string][]MiddlewareWithPath, 10),
		orderCounter: 0,
		radixRouter: router.New(),
	}
}

// Use 为指定的HTTP方法和路径注册中间件
func (r *Router) Use(method string, path string, m Middleware) {
	// 如果没有指定方法，则默认注册所有方法
	if method == "" {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
		for _, method := range methods {
			r.Use(method, path, m)
		}
		return
	}

	// 如果没有对应的中间件列表，则创建一个
	if _, ok := r.middlewares[method]; !ok {
		r.middlewares[method] = make([]MiddlewareWithPath, 0, 10)
	}

	// 根据路径类型分类
	mwType := classifyMiddlewareType(path)

	r.orderCounter++

	// 根据路径类型确定中间件的来源
	var source MiddlewareSource
	if path == "/*" {
		source = GlobalSource
	} else {
		source = PathSource
	}

	mwWithPath := MiddlewareWithPath{
		Middleware: m,
		Path:       path,
		Type:       mwType,
		Order:      r.orderCounter,
		Source:     source,
	}

	r.middlewares[method] = append(r.middlewares[method], mwWithPath)
}

// findMatchedNodes 查找匹配的节点，用于向后兼容
func (r *Router) findMatchedNodes(method string, path string) []*node {
	// 这个方法仅用于向后兼容，实际不会被调用
	return nil
}

// getChildren 获取所有子节点，用于向后兼容
func (r *Router) getChildren(n *node) []*node {
	// 这个方法仅用于向后兼容，实际不会被调用
	return nil
}

// Get 注册GET方法路由
func (r *Router) Get(path string, handlerFunc HandlerFunc) {
	r.addHandler("GET", path, handlerFunc)
}

// Post 注册POST方法路由
func (r *Router) Post(path string, handlerFunc HandlerFunc) {
	r.addHandler("POST", path, handlerFunc)
}

// Put 注册PUT方法路由
func (r *Router) Put(path string, handlerFunc HandlerFunc) {
	r.addHandler("PUT", path, handlerFunc)
}

// Delete 注册DELETE方法路由
func (r *Router) Delete(path string, handlerFunc HandlerFunc) {
	r.addHandler("DELETE", path, handlerFunc)
}

// Patch 注册PATCH方法路由
func (r *Router) Patch(path string, handlerFunc HandlerFunc) {
	r.addHandler("PATCH", path, handlerFunc)
}

// Options 注册OPTIONS方法路由
func (r *Router) Options(path string, handlerFunc HandlerFunc) {
	r.addHandler("OPTIONS", path, handlerFunc)
}

// addHandler 注册路由处理函数
func (r *Router) addHandler(method string, path string, handlerFunc HandlerFunc) {
	// 路由校验
	if path == "" {
		panic("path cannot be empty")
	}

	if path[0] != '/' {
		panic("path must begin with /")
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		panic("path must not end with /")
	}

	// 检查是否包含连续的斜杠
	if strings.Contains(path, "//") {
		panic("path cannot contain //")
	}

	// 使用新的RadixTree路由器添加路由
	r.radixRouter.Handle(method, path, handlerFunc)
}

// findHandler 查找路由处理函数
func (r *Router) findHandler(method string, path string, ctx *Context) (*node, bool) {
	// 初始化参数映射
	params := router.AcquireParams()
	defer router.ReleaseParams(params)

	// 使用新的RadixTree查找路由处理函数
	handler, ok := r.radixRouter.Find(method, path, params)
	if !ok {
		return nil, false
	}

	// 将找到的路径参数复制到上下文中
	for k, v := range params {
		ctx.Param[k] = v
	}

	// 创建一个临时节点，模拟原路由系统的节点结构
	tempNode := &node{
		path:    path,
		handler: handler.(HandlerFunc),
	}

	return tempNode, true
}