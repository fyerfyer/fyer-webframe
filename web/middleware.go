package web

import (
	"sort"
	"strings"
)

// HandlerFunc 定义请求处理函数
type HandlerFunc func(ctx *Context)

// ErrorHandlerFunc 定义带错误返回的请求处理函数
type ErrorHandleFunc func(ctx *Context) error

// Middleware 定义中间件函数
type Middleware func(HandlerFunc) HandlerFunc

// ErrorHandlingMiddleware 定义带错误处理的中间件函数
type ErrorHandlingMiddleware func(ErrorHandleFunc) ErrorHandleFunc

// MiddlewareSource 表示中间件的注册来源
type MiddlewareSource int

const (
	// GlobalSource 表示全局中间件
	GlobalSource MiddlewareSource = iota
	// PathSource 表示路径特定中间件
	PathSource
)

// MiddlewareType 表示中间件的类型
type MiddlewareType int

const (
	StaticMiddleware MiddlewareType = iota
	RegexMiddleware
	ParamMiddleware
	WildcardMiddleware
)

// MiddlewareWithPath 存储中间件及其路径信息
type MiddlewareWithPath struct {
	Middleware Middleware
	Path       string
	Type       MiddlewareType
	Order      int
	Source     MiddlewareSource
}

// WithErrorHandling 将中间件转换为带错误处理的中间件
func WithErrorHandling(m Middleware) ErrorHandlingMiddleware {
	return func(next ErrorHandleFunc) ErrorHandleFunc {
		return func(ctx *Context) error {
			var err error
			m(func(c *Context) {
				err = next(c)
			})(ctx)
			return err
		}
	}
}

// classifyMiddlewareType 根据路径类型分类中间件
func classifyMiddlewareType(path string) MiddlewareType {
	// Static paths don't have special characters
	if !strings.Contains(path, ":") && !strings.Contains(path, "*") {
		return StaticMiddleware
	}

	// 检查正则路由
	if strings.Contains(path, "(") && strings.Contains(path, ")") {
		return RegexMiddleware
	}

	// 检查参数路由
	if strings.Contains(path, ":") {
		return ParamMiddleware
	}

	// 检查通配符路由
	if strings.Contains(path, "*") {
		return WildcardMiddleware
	}

	return StaticMiddleware
}

// pathMatchesStaticPattern 检验静态路由匹配
func pathMatchesStaticPattern(reqPath, middlewarePath string) bool {
	// 如果路径相同，直接匹配
	if reqPath == middlewarePath {
		return true
	}

	if middlewarePath == "/" {
		return true
	}

	// 如果中间件路径没有通配符，考虑只匹配一部分
	// 例如 /users/profile 可以匹配 /users 中间件
	if !strings.Contains(middlewarePath, "*") && !strings.Contains(middlewarePath, ":") {
		return strings.HasPrefix(reqPath, middlewarePath+"/") || reqPath == middlewarePath
	}

	return false
}

// pathMatchesParamPattern 检验参数路径匹配
func pathMatchesParamPattern(reqPath, patternPath string) bool {
	// 如果pattern没有参数路径，直接返回
	if !strings.Contains(patternPath, ":") {
		return false
	}

	reqSegments := strings.Split(strings.Trim(reqPath, "/"), "/")
	patternSegments := strings.Split(strings.Trim(patternPath, "/"), "/")

	// 检查路径段数
	if len(reqSegments) != len(patternSegments) {
		return false
	}

	for i, segment := range patternSegments {
		// 参数路径不需要匹配
		if strings.HasPrefix(segment, ":") {
			continue
		}

		// 非参数路径需要匹配
		if segment != reqSegments[i] {
			return false
		}
	}

	return true
}

// pathMatchesRegexPattern 检查正则参数路径匹配
func pathMatchesRegexPattern(reqPath, patternPath string) bool {
	// 首先检查是否为参数路径
	if !pathMatchesParamPattern(reqPath, patternPath) {
		return false
	}

	// 如果路径中包含正则表达式，直接返回
	// 实际的正则匹配已经在路由解析中完成
	return strings.Contains(patternPath, "(") && strings.Contains(patternPath, ")")
}

// pathMatchesWildcardPattern 检查通配符路径匹配
func pathMatchesWildcardPattern(reqPath, wildcardPath string) bool {
	// 通配符匹配所有路径
	if wildcardPath == "/*" {
		return true
	}

	// 去掉末尾的通配符
	// 根据规定，通配符只存在于末尾
	if strings.HasSuffix(wildcardPath, "/*") {
		basePath := wildcardPath[:len(wildcardPath)-2]
		// 检查是否为根路径
		return reqPath == basePath || strings.HasPrefix(reqPath, basePath+"/")
	}

	return true
}

// collectMatchingMiddlewares 返回所有符合所给路径的中间件
func collectMatchingMiddlewares(middlewares []MiddlewareWithPath, actualPath string) []MiddlewareWithPath {
	var matchingMiddlewares []MiddlewareWithPath

	for _, mw := range middlewares {
		var matches bool

		switch mw.Type {
		case StaticMiddleware:
			matches = pathMatchesStaticPattern(actualPath, mw.Path)
		case RegexMiddleware:
			matches = pathMatchesRegexPattern(actualPath, mw.Path)
		case ParamMiddleware:
			matches = pathMatchesParamPattern(actualPath, mw.Path)
		case WildcardMiddleware:
			matches = pathMatchesWildcardPattern(actualPath, mw.Path)
		}

		if matches {
			matchingMiddlewares = append(matchingMiddlewares, mw)
		}
	}

	return matchingMiddlewares
}

// calculatePathSpecificity 为路径计算特定性分数
// 分数越高越具体，越先匹配
func calculatePathSpecificity(path string) int {
	// 去除前导 /
	path = strings.TrimPrefix(path, "/")

	segments := strings.Split(path, "/")

	// 利用连续静态路径匹配数量来决定分数
	var score int = 0
	var continuousStaticBonus int = 1000000 // 连续静态路径分数
	var currentContinuousCount int = 0
	var segmentPositionValue int = 100000   // 每个字段的分数

	// 首先优先考虑连续静态路径
	for _, segment := range segments {
		if !strings.Contains(segment, ":") && !strings.Contains(segment, "*") {
			currentContinuousCount++
			score += continuousStaticBonus * currentContinuousCount
		} else {
			// 在第一个非静态匹配路径上终止
			break
		}
	}

	// 其次考虑路径段位置
	// 越前面的路径段分数越高
	for i, segment := range segments {
		if !strings.Contains(segment, ":") && !strings.Contains(segment, "*") {
			// 只为不连续的静态路径加分
			if i >= currentContinuousCount {
				score += segmentPositionValue / (i + 1)
			}
		}
	}

	// 然后考虑静态、正则、参数和通配符路径类型
	// 计算各种类型路径段的数量
	staticCount := 0
	regexCount := 0
	paramCount := 0
	wildcardCount := 0

	for _, segment := range segments {
		if strings.Contains(segment, "*") {
			wildcardCount++
		} else if strings.Contains(segment, "(") && strings.Contains(segment, ")") {
			regexCount++
		} else if strings.Contains(segment, ":") {
			paramCount++
		} else {
			staticCount++
		}
	}

	// 为不同的路径段类型添加权重
	score += staticCount * 1000
	score += regexCount * 100
	score += paramCount * 10
	score += wildcardCount * 1

	// 最后考虑路径长度
	score += len(path)

	return score
}

// sortMiddlewares 基于以下原则排序：
// 1. 首先按来源类型：GlobalSource最先执行
// 2. 然后按照具体性分数来排序
// 3. 最后按照先后顺序排序
func sortMiddlewares(middlewares []MiddlewareWithPath) []MiddlewareWithPath {
	// 复制一份，不修改原有的中间件列表
	result := make([]MiddlewareWithPath, len(middlewares))
	copy(result, middlewares)

	// 根据前面的优先级顺序进行排序
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Source != result[j].Source {
			return result[i].Source < result[j].Source
		}

		if result[i].Source == PathSource {
			if result[i].Type != result[j].Type {
				return result[i].Type < result[j].Type
			}

			specI := calculatePathSpecificity(result[i].Path)
			specJ := calculatePathSpecificity(result[j].Path)
			if specI != specJ {
				return specI > specJ
			}
		}

		return result[i].Order < result[j].Order
	})

	return result
}

// BuildChain 构建中间件执行链
func BuildChain(handler HandlerFunc, actualPath string, middlewares []MiddlewareWithPath) HandlerFunc {
	matchingMiddlewares := collectMatchingMiddlewares(middlewares, actualPath)
	sortedMiddlewares := sortMiddlewares(matchingMiddlewares)

	for i := len(sortedMiddlewares) - 1; i >= 0; i-- {
		handler = sortedMiddlewares[i].Middleware(handler)
	}

	return func(ctx *Context) {
		ctx.aborted = false

		if !ctx.IsAborted() {
			handler(ctx)
		}
	}
}