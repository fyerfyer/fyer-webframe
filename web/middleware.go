package web

import (
	"sort"
	"strings"
)

type HandlerFunc func(ctx *Context)

type ErrorHandleFunc func(ctx *Context) error

type Middleware func(HandlerFunc) HandlerFunc

type ErrorHandlingMiddleware func(ErrorHandleFunc) ErrorHandleFunc

// MiddlewareWithPath 存储中间件及其对应的路径信息
type MiddlewareWithPath struct {
	Middleware Middleware
	Path       string
}

// WithErrorHandling 包装标准中间件来支持错误发送功能
func WithErrorHandling(m Middleware) ErrorHandlingMiddleware {
	return func(next ErrorHandleFunc) ErrorHandleFunc {
		return func(ctx *Context) error {
			var err error
			m(func(c *Context) {
				if e := next(c); e != nil {
					err = e
				}
			})(ctx)
			return err
		}
	}
}

// sortBySpecificity 根据路径的特异性对中间件进行排序
// 路径越长越具体，应该优先执行
func sortBySpecificity(middlewares []MiddlewareWithPath) []Middleware {
	// 根据路径长度排序，长度越长，优先级越高
	sort.Slice(middlewares, func(i, j int) bool {
		// 移除前导斜杠以确保公平比较
		pathI := strings.TrimPrefix(middlewares[i].Path, "/")
		pathJ := strings.TrimPrefix(middlewares[j].Path, "/")

		// 首先比较路径段数
		segmentsI := strings.Count(pathI, "/") + 1
		segmentsJ := strings.Count(pathJ, "/") + 1

		if segmentsI != segmentsJ {
			return segmentsI > segmentsJ // 路径段数更多的优先
		}

		// 如果路径段数相同，比较整体路径长度
		return len(pathI) > len(pathJ) // 路径更长的优先
	})

	// 提取排序后的中间件函数
	result := make([]Middleware, len(middlewares))
	for i, m := range middlewares {
		result[i] = m.Middleware
	}

	return result
}

// BuildChain 构建中间件执行链，按照静态->正则->参数->通配符的顺序
// 同时在每种类型内部，按照路径特异性排序（更具体的路径优先执行）
func BuildChain(n *node, handler HandlerFunc) HandlerFunc {
	// 对每种类型的中间件按照路径特异性排序
	sortedStatic := sortBySpecificity(n.staticMiddlewares)
	sortedRegex := sortBySpecificity(n.regexMiddlewares)
	sortedParam := sortBySpecificity(n.paramMiddlewares)
	sortedWildcard := sortBySpecificity(n.wildcardMiddlewares)

	// 通配符中间件（最后执行）
	for i := len(sortedWildcard) - 1; i >= 0; i-- {
		handler = sortedWildcard[i](handler)
	}

	// 参数路由中间件
	for i := len(sortedParam) - 1; i >= 0; i-- {
		handler = sortedParam[i](handler)
	}

	// 正则路由中间件
	for i := len(sortedRegex) - 1; i >= 0; i-- {
		handler = sortedRegex[i](handler)
	}

	// 静态路由中间件（最先执行）
	for i := len(sortedStatic) - 1; i >= 0; i-- {
		handler = sortedStatic[i](handler)
	}

	return handler
}