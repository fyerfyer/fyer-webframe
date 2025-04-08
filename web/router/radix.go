package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// RadixTree 实现基于压缩前缀树的路由匹配
type RadixTree struct {
	// 按HTTP方法存储的路由树
	trees map[string]*Node
	// 用于保护树的并发修改
	mu sync.RWMutex
}

// NewRadixTree 创建一个新的RadixTree实例
func NewRadixTree() *RadixTree {
	return &RadixTree{
		trees: make(map[string]*Node),
	}
}

// Add 添加一个新的路由到对应HTTP方法的树中
func (r *RadixTree) Add(method, path string, handler interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果当前HTTP方法还没有对应的树，则创建一个
	if _, ok := r.trees[method]; !ok {
		r.trees[method] = NewNode()
	}

	// 将路由添加到对应的树中
	r.trees[method].Insert(path, handler)
}

// Find 查找给定路径的处理函数
func (r *RadixTree) Find(method, path string, params map[string]string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 检查当前HTTP方法是否有对应的树
	root, ok := r.trees[method]
	if (!ok) {
		return nil, false
	}

	// 使用树查找对应的处理函数
	return root.Find(path, params)
}

// 为常用HTTP方法提供便捷方法

// GET 注册一个GET方法的路由
func (r *RadixTree) GET(path string, handler interface{}) {
	r.Add(http.MethodGet, path, handler)
}

// POST 注册一个POST方法的路由
func (r *RadixTree) POST(path string, handler interface{}) {
	r.Add(http.MethodPost, path, handler)
}

// PUT 注册一个PUT方法的路由
func (r *RadixTree) PUT(path string, handler interface{}) {
	r.Add(http.MethodPut, path, handler)
}

// DELETE 注册一个DELETE方法的路由
func (r *RadixTree) DELETE(path string, handler interface{}) {
	r.Add(http.MethodDelete, path, handler)
}

// PATCH 注册一个PATCH方法的路由
func (r *RadixTree) PATCH(path string, handler interface{}) {
	r.Add(http.MethodPatch, path, handler)
}

// OPTIONS 注册一个OPTIONS方法的路由
func (r *RadixTree) OPTIONS(path string, handler interface{}) {
	r.Add(http.MethodOptions, path, handler)
}

// HEAD 注册一个HEAD方法的路由
func (r *RadixTree) HEAD(path string, handler interface{}) {
	r.Add(http.MethodHead, path, handler)
}

// PrintTree 打印路由树结构，用于调试目的
func (r *RadixTree) PrintTree() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("RadixTree Routes:\n")

	for method, root := range r.trees {
		sb.WriteString(fmt.Sprintf("Method: %s\n", method))
		printNode(&sb, root, 0)
		sb.WriteString("\n")
	}

	return sb.String()
}

// printNode 递归打印节点，用于调试
func printNode(sb *strings.Builder, n *Node, level int) {
	indent := strings.Repeat("  ", level)

	sb.WriteString(fmt.Sprintf("%s- %s", indent, n.path))
	if n.handler != nil {
		sb.WriteString(" [Handler]")
	}
	if n.isParam {
		sb.WriteString(fmt.Sprintf(" [Param: %s]", n.paramName))
	}
	if n.isRegex {
		sb.WriteString(fmt.Sprintf(" [Regex: %s]", n.pattern.String()))
	}
	sb.WriteString("\n")

	// 打印子节点
	for _, child := range n.children {
		printNode(sb, child, level+1)
	}

	// 打印参数子节点
	if n.paramChild != nil {
		printNode(sb, n.paramChild, level+1)
	}

	// 打印正则子节点
	for _, regexChild := range n.regexChildren {
		printNode(sb, regexChild, level+1)
	}

	// 打印通配符子节点
	if n.wildcardChild != nil {
		printNode(sb, n.wildcardChild, level+1)
	}
}

// Routes 返回所有注册的路由数量
func (r *RadixTree) Routes() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int
	for _, root := range r.trees {
		count += countHandlers(root)
	}
	return count
}

// countHandlers 统计节点中的处理函数数量
func countHandlers(n *Node) int {
	if n == nil {
		return 0
	}

	count := 0
	if n.handler != nil {
		count = 1
	}

	// 统计子节点的处理函数
	for _, child := range n.children {
		count += countHandlers(child)
	}

	// 统计参数子节点
	if n.paramChild != nil {
		count += countHandlers(n.paramChild)
	}

	// 统计正则子节点
	for _, regexChild := range n.regexChildren {
		count += countHandlers(regexChild)
	}

	// 统计通配符子节点
	if n.wildcardChild != nil {
		count += countHandlers(n.wildcardChild)
	}

	return count
}