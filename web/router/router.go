package router

// Router 使用RadixTree实现的HTTP路由器
type Router struct {
	// 使用RadixTree进行路由存储和匹配
	tree *RadixTree
}

// New 创建一个新的路由器实例
func New() *Router {
	return &Router{
		tree: NewRadixTree(),
	}
}

// GET 注册GET方法的路由处理函数
func (r *Router) GET(path string, handler interface{}) {
	r.tree.GET(path, handler)
}

// POST 注册POST方法的路由处理函数
func (r *Router) POST(path string, handler interface{}) {
	r.tree.POST(path, handler)
}

// PUT 注册PUT方法的路由处理函数
func (r *Router) PUT(path string, handler interface{}) {
	r.tree.PUT(path, handler)
}

// DELETE 注册DELETE方法的路由处理函数
func (r *Router) DELETE(path string, handler interface{}) {
	r.tree.DELETE(path, handler)
}

// PATCH 注册PATCH方法的路由处理函数
func (r *Router) PATCH(path string, handler interface{}) {
	r.tree.PATCH(path, handler)
}

// OPTIONS 注册OPTIONS方法的路由处理函数
func (r *Router) OPTIONS(path string, handler interface{}) {
	r.tree.OPTIONS(path, handler)
}

// HEAD 注册HEAD方法的路由处理函数
func (r *Router) HEAD(path string, handler interface{}) {
	r.tree.HEAD(path, handler)
}

// Handle 是一个通用的路由注册方法，可以指定HTTP方法
func (r *Router) Handle(method, path string, handler interface{}) {
	r.tree.Add(method, path, handler)
}

// Find 根据HTTP方法和路径查找处理函数
func (r *Router) Find(method, path string, params map[string]string) (interface{}, bool) {
	return r.tree.Find(method, path, params)
}

// Routes 返回路由器中注册的路由数量
func (r *Router) Routes() int {
	return r.tree.Routes()
}

// PrintRoutes 返回路由树的字符串表示，用于调试
func (r *Router) PrintRoutes() string {
	return r.tree.PrintTree()
}