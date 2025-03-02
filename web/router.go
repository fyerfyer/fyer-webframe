package web

import (
	"regexp"
	"strings"
)

type Router struct {
	routerTrees map[string]*node
	middlewares map[string][]MiddlewareWithPath // 使用http方法作为键值对
	orderCounter int                            // 用于记录中间件注册顺序
}

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
	regexPattern        *regexp.Regexp
	staticMiddlewares   []MiddlewareWithPath
	regexMiddlewares    []MiddlewareWithPath
	paramMiddlewares    []MiddlewareWithPath
	wildcardMiddlewares []MiddlewareWithPath
}

func NewRouter() *Router {
	return &Router{
		routerTrees: make(map[string]*node),
		middlewares: make(map[string][]MiddlewareWithPath, 10),
		orderCounter: 0,
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
		r.middlewares[method] = make([]MiddlewareWithPath, 0)
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

// findMatchedNodes 查找匹配的节点
func (r *Router) findMatchedNodes(method string, path string) []*node {
	var matched []*node
	root := r.routerTrees[method]

	// 处理根路径特殊情况
	if path == "/" {
		if root.handler != nil {
			matched = append(matched, root)
		}
		return matched
	}

	if path == "/*" {
		return r.getChildren(root)
	}

	// 标准路径处理
	path = path[1:]
	segments := strings.Split(path, "/")

	var findMatched func(*node, []string)
	findMatched = func(n *node, segments []string) {
		// 如果到达路径末端
		if len(segments) == 0 {
			if n.handler != nil {
				matched = append(matched, n)
			}
			return
		}

		segment := segments[0]

		// 处理通配符路径
		if segment == "*" {
			// 遍历父节点的所有子节点，因为通配符可以匹配任何路径，则只要是父节点的子节点就可以匹配
			matched = append(matched, r.getChildren(n)...)
		}

		// 处理参数路径
		// 这里处理的情况是：中间件路由和查找节点路由都是参数路由的格式
		// 中间件路由为参数路由、查找结点路由为具体的字符串的这一情况在后面讨论
		if segment[0] == ':' {
			// 遍历父节点的所有子节点，参数路径这一段子节点可以忽略，但是其他的部分还要继续匹配
			paramSegment := segment
			children := r.getChildren(n.parent)
			for _, child := range children {
				// 首先先检验是否为正则匹配，如果有正则匹配的话要确保匹配成功
				// 判断正则路由格式是否正确
				hasLeftParenthesis := strings.Contains(paramSegment, "(")
				hasRightParenthesis := strings.HasSuffix(paramSegment, ")")
				middleHasRegex := hasLeftParenthesis || hasRightParenthesis

				// 如果中间件路由是带有正则表达式的，那么当前节点必须包含相同的正则表达式才能匹配
				if middleHasRegex && !child.isRegex {
					continue
				}

				if middleHasRegex {
					parts := strings.SplitN(paramSegment, "(", 2)
					paramSegment = parts[0][1:]
					pattern := parts[1][:len(parts[1])-1]

					// 判断正则表达式是否合法
					if !child.regexPattern.MatchString(pattern) {
						continue
					}
				}

				if child.handler != nil {
					matched = append(matched, child)
				}

				if child.children != nil {
					for _, c := range child.children {
						findMatched(c, segments[1:])
					}
				}
			}
		} else {
			// 查找结点路由为具体的字符串的情况
			if paramNodes, ok, _ := n.getParamChild(); ok {
				for _, paramNode := range paramNodes {
					if paramNode.isRegex {
						if paramNode.regexPattern.MatchString(segment) {
							findMatched(paramNode, segments[1:])
						}
					} else {
						findMatched(paramNode, segments[1:])
					}
				}
			}
		}

		// 检查静态匹配
		if child, ok := n.children[segment]; ok {
			findMatched(child, segments[1:])
		}

		// 检查通配符匹配
		if wildcardNode, ok := n.children["*"]; ok {
			if wildcardNode.handler != nil {
				matched = append(matched, wildcardNode)
			}
		}
	}

	findMatched(root, segments)
	return matched
}

// getChildren 获取所有子节点
func (r *Router) getChildren(n *node) []*node {
	var children []*node
	if n.children == nil {
		return nil
	}
	for _, child := range n.children {
		if child.handler != nil {
			children = append(children, child)
		}
		if nodes := r.getChildren(child); nodes != nil {
			children = append(children, nodes...)
		}
	}
	return children
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
	// 首先查看是否有对应根节点，没有的话就创建一个
	if _, ok := r.routerTrees[method]; !ok {
		r.routerTrees[method] = &node{
			path: "/",
		}
	}

	root := r.routerTrees[method]

	// 根节点特判：'/'
	if path == "/" {
		if root.handler != nil {
			panic("handler already exists: " + path)
		}
		root.handler = handlerFunc
		return
	}

	// 路由校验
	if path == "" {
		panic("path cannot be empty")
	}

	if path[0] != '/' {
		panic("path must begin with /")
	}

	if path[len(path)-1] == '/' {
		panic("path must not end with /")
	}

	// 路径分割，逐级创建子节点
	path = path[1:]
	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if segment == "" {
			panic("path cannot have empty segment")
		}

		// 只允许单个通配符，并且通配符后不应有其他路径
		if root.hasStarParam {
			panic("* must be the last segment")
		}

		// 通配符和参数路径不能同时存在
		if segment == "*" {
			if root.hasParamChild {
				panic("cannot have * and param in the same path")
			}
		}

		// 创建子节点，然后进入下一级
		isParam := false
		isRegex := false
		var regexPattern *regexp.Regexp

		if segment[0] == ':' {
			segment = segment[1:]
			// 检查是否为参数路径
			if strings.Contains(segment, "(") && strings.HasSuffix(segment, ")") {
				isRegex = true
				// 解析正则表达式
				parts := strings.SplitN(segment, "(", 2)
				segment = parts[0]
				pattern := parts[1][:len(parts[1])-1] // 去掉后缀')'
				var err error
				regexPattern, err = regexp.Compile(pattern)
				if err != nil {
					panic("invalid regex pattern: " + err.Error())
				}
			}
			isParam = true

			// 参数路径和通配符不能同时存在
			if root.hasStarChild {
				panic("cannot have * and param in the same path")
			}

			root.hasParamChild = true
		}

		root.createChild(segment, isRegex, regexPattern)
		root = root.children[segment]
		root.isParam = isParam
		root.isRegex = isRegex
		root.regexPattern = regexPattern
	}

	// 最后一个节点设置 handler
	// 如果设置了重复的 handler，说明重复注册了路由
	if root.handler != nil {
		panic("handler already exists: " + path)
	}

	if _, ok, isLast := root.parent.getParamChild(); ok && isLast {
		panic("cannot register more than one param in the same path")
	}

	root.handler = handlerFunc
	return
}

// findHandler 查找路由处理函数
func (r *Router) findHandler(method string, path string, ctx *Context) (*node, bool) {
	if ctx.Param == nil {
		ctx.Param = make(map[string]string)
	}

	if _, ok := r.routerTrees[method]; !ok {
		return nil, false
	}

	root := r.routerTrees[method]
	if path == "/" {
		return root, true
	}

	if path == "" {
		return nil, false
	}

	if path[0] != '/' {
		return nil, false
	}

	if path[len(path)-1] == '/' {
		return nil, false
	}

	path = path[1:]
	segments := strings.Split(path, "/")

	var wildcard *node
	remainingSegments := segments
	var matchParamNode func(root *node, segments []string, tempCtx *Context) (*node, bool)
	matchParamNode = func(root *node, segments []string, tempCtx *Context) (*node, bool) {
		// 1. 检查是否到达路径末端
		if len(segments) == 0 {
			// 只有当前节点有handler时才算匹配成功
			if root.handler != nil {
				return root, true
			}
			return nil, false
		}

		segment := segments[0]
		staticNode, wildcardNode, paramNodes, staticOK, wildcardOK, paramOK := root.findChild(segment)

		// 2. 优先尝试静态匹配
		if staticOK {
			if node, ok := matchParamNode(staticNode, segments[1:], tempCtx); ok {
				return node, true
			}
		}

		// 3. 尝试参数匹配
		if paramOK {
			for _, paramNode := range paramNodes {
				newTempCtx := &Context{Param: make(map[string]string)}
				for k, v := range tempCtx.Param {
					newTempCtx.Param[k] = v
				}
				newTempCtx.Param[paramNode.path] = segment

				if matchedNode, ok := matchParamNode(paramNode, segments[1:], newTempCtx); ok {
					// 找到完整匹配，更新实际的context
					for k, v := range newTempCtx.Param {
						ctx.Param[k] = v
					}
					return matchedNode, true
				}
			}
		}

		// 4. 如果有通配符匹配，则直接匹配即可，后面的不需要管
		if wildcardOK {
			return wildcardNode, true
		}

		return nil, false
	}

	tempCtx := &Context{Param: make(map[string]string)}
	if node, ok := matchParamNode(root, remainingSegments, tempCtx); ok {
		if node.Param == nil {
			node.Param = make(map[string]string)
		}
		node.Param = ctx.Param
		return node, true
	}

	return wildcard, false
}

// createChild 创建子节点
func (n *node) createChild(path string, isRegex bool, pattern *regexp.Regexp) {
	if n.children == nil {
		n.children = make(map[string]*node)
	}

	// 通配符匹配：只能带*
	isStar := false
	if path[0] == '*' {
		if len(path) != 1 {
			panic("only one * is allowed in path")
		}
		isStar = true
		n.hasStarChild = true
	}

	if _, ok := n.children[path]; !ok {
		n.children[path] = &node{
			path:         path,
			hasStarParam: isStar,
			parent:       n,
			isRegex:      isRegex,
			regexPattern: pattern,
		}
	}
}

// findChild 查找子节点
func (n *node) findChild(path string) (*node, *node, []*node, bool, bool, bool) {
	if n.children == nil {
		return nil, nil, nil, false, false, false
	}

	staticNode, staticOK := n.children[path]
	wildcardNode, wildcardOK := n.children["*"]
	paramNodes, paramOK, _ := n.getParamChild()

	// 筛选参数节点，只包括那些匹配正则表达式模式的节点
	if paramOK {
		var matchingParamNodes []*node
		for _, paramNode := range paramNodes {
			if paramNode.isRegex {
				if paramNode.regexPattern.MatchString(path) {
					matchingParamNodes = append(matchingParamNodes, paramNode)
				}
			} else {
				matchingParamNodes = append(matchingParamNodes, paramNode)
			}
		}
		paramNodes = matchingParamNodes
		paramOK = len(paramNodes) > 0
	}

	return staticNode, wildcardNode, paramNodes, staticOK, wildcardOK, paramOK
}

// getParamChild 获取参数节点
func (n *node) getParamChild() ([]*node, bool, bool) {
	if n.children == nil {
		return nil, false, false
	}

	var paramNodes []*node
	hasParamChild := false
	hasParamHandler := false
	for _, child := range n.children {
		if child.isParam {
			paramNodes = append(paramNodes, child)
			hasParamChild = true
			if child.handler != nil {
				hasParamHandler = true
			}
		}
	}

	return paramNodes, hasParamChild, hasParamHandler
}