package web

type Middleware func(HandlerFunc) HandlerFunc

// BuildChain 构建中间件执行链，按照静态->正则->参数->通配符的顺序
func BuildChain(n *node, handler HandlerFunc) HandlerFunc {
	// 添加Resp写回中间件

	// 通配符中间件（最后执行）
	for i := len(n.wildcardMiddlewares) - 1; i >= 0; i-- {
		handler = n.wildcardMiddlewares[i](handler)
	}

	// 参数路由中间件
	for i := len(n.paramMiddlewares) - 1; i >= 0; i-- {
		handler = n.paramMiddlewares[i](handler)
	}

	// 正则路由中间件
	for i := len(n.regexMiddlewares) - 1; i >= 0; i-- {
		handler = n.regexMiddlewares[i](handler)
	}

	// 静态路由中间件（最先执行）
	for i := len(n.staticMiddlewares) - 1; i >= 0; i-- {
		handler = n.staticMiddlewares[i](handler)
	}

	return handler
}
