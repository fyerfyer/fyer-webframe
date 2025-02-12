package web

import (
	"strings"
)

type Router struct {
	routerTrees map[string]*node
}

type node struct {
	path         string
	children     map[string]*node
	handler      HandlerFunc
	hasStarParam bool
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

	// 根节点特判：'/'或者'*'
	// 但是不支持单独的'/*'
	if path == "/" || path == "*" {
		if root.handler != nil {
			panic("handler already exists: " + path)
		}
		root.handler = handlerFunc
		root.hasStarParam = path == "*" // 通配符标记
		return
	}

	if path == "/*" {
		panic("cannot register router of '/*' type, please use * instead")
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

		// 创建子节点，然后进入下一级
		root.createChild(segment)
		root = root.children[segment]
	}

	// 最后一个节点设置 handler
	// 如果设置了重复的 handler，说明重复注册了路由
	if root.handler != nil {
		panic("handler already exists: " + path)
	}

	root.handler = handlerFunc
	return
}

func (r *Router) findHandler(method string, path string) (*node, bool) {
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
	for _, segment := range segments {
		if segment == "" {
			return nil, false
		}

		if segment == "*" {
			if !root.hasStarParam {
				return nil, false
			}

			return root, true
		}

		// 如果找不到，就选择通配符匹配，否则优先静态匹配
		staticNode, wildcardNode, staticOK, wildcardOK := root.findChild(segment)

		// 首先更新已有的通配符节点，这样只要遇到了静态匹配无法处理的就可以返回通配符节点
		if wildcardOK {
			wildcard = wildcardNode
		}

		if !staticOK {
			return wildcard, wildcard != nil
		}

		root = staticNode
	}

	return root, true
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
	}

	if _, ok := n.children[path]; !ok {
		n.children[path] = &node{
			path:         path,
			hasStarParam: isStar,
		}
	}
}

func (n *node) findChild(path string) (*node, *node, bool, bool) {
	if n.children == nil {
		return nil, nil, false, false
	}

	staticNode, staticOK := n.children[path]
	wildcardNode, wildcardOK := n.children["*"]

	return staticNode, wildcardNode, staticOK, wildcardOK
}
