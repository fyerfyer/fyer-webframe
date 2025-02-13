package web

import (
	"strings"
)

type Router struct {
	routerTrees map[string]*node
}

type node struct {
	path          string
	children      map[string]*node
	parent        *node
	handler       HandlerFunc
	hasStarParam  bool
	isParam       bool
	hasParamChild bool
	hasStarChild  bool
	Param         map[string]string
}

func NewRouter() *Router {
	return &Router{
		routerTrees: make(map[string]*node),
	}
}

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
		if segment[0] == ':' {
			segment = segment[1:]
			isParam = true

			// 参数路径和通配符不能同时存在
			if root.hasStarChild {
				panic("cannot have * and param in the same path")
			}

			root.hasParamChild = true
		}

		root.createChild(segment)
		root = root.children[segment]
		root.isParam = isParam
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
		if len(segments) == 0 {
			return root, true
		}

		segment := segments[0]
		staticNode, wildcardNode, paramNodes, staticOK, wildcardOK, paramOK := root.findChild(segment)

		// 优先尝试静态匹配
		if staticOK {
			return matchParamNode(staticNode, segments[1:], tempCtx)
		}

		// 尝试参数匹配
		if paramOK {
			// 对每个参数节点尝试匹配
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

		// 如果有通配符匹配，保存但继续尝试其他匹配
		if wildcardOK {
			wildcard = wildcardNode
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

	return wildcard, wildcard != nil
}

func (n *node) createChild(path string) {
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
		}
	}
}

func (n *node) findChild(path string) (*node, *node, []*node, bool, bool, bool) {
	if n.children == nil {
		return nil, nil, nil, false, false, false
	}

	staticNode, staticOK := n.children[path]
	wildcardNode, wildcardOK := n.children["*"]
	paramNodes, paramOK, _ := n.getParamChild()

	return staticNode, wildcardNode, paramNodes, staticOK, wildcardOK, paramOK
}

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
			hasParamHandler = child.handler != nil
		}
	}

	return paramNodes, hasParamChild, hasParamHandler
}
