package router

import (
    "regexp"
    "strings"
)

// Node 表示Radix Tree中的一个节点
type Node struct {
    // 节点路径片段
    path string

    // 静态子节点映射
    children map[string]*Node

    // 参数子节点，例如 :id
    paramChild *Node

    // 通配符子节点，例如 *
    wildcardChild *Node

    // 处理函数
    handler interface{}

    // 是否是参数节点
    isParam bool

    // 参数名称，例如在 :id 中，参数名为 "id"
    paramName string

    // 是否是正则节点
    isRegex bool

    // 正则表达式
    pattern *regexp.Regexp
}

// NewNode 创建并返回一个新的节点
func NewNode() *Node {
    return &Node{
        children: make(map[string]*Node),
    }
}

// Insert 将路径和对应的处理函数插入到Radix Tree中
func (n *Node) Insert(path string, handler interface{}) {
    // 处理根路径特殊情况
    if path == "/" {
        n.handler = handler
        return
    }

    // 标准化路径格式
    path = strings.Trim(path, "/")
    segments := strings.Split(path, "/")

    current := n
    for i, segment := range segments {
        // 跳过空段
        if segment == "" {
            continue
        }

        // 根据段类型处理节点
        if segment == "*" {
            // 通配符节点
            if current.wildcardChild == nil {
                current.wildcardChild = &Node{
                    path:     "*",
                    children: make(map[string]*Node),
                }
            }
            current = current.wildcardChild
        } else if segment[0] == ':' {
            // 参数节点或正则节点
            paramName := segment[1:]
            isRegex := false
            var pattern *regexp.Regexp

            // 提取正则表达式
            if strings.Contains(paramName, "(") && strings.Contains(paramName, ")") {
                regexStart := strings.Index(paramName, "(")
                regexEnd := strings.LastIndex(paramName, ")")

                if regexStart > 0 && regexEnd > regexStart {
                    regexStr := paramName[regexStart+1:regexEnd]
                    paramName = paramName[:regexStart]

                    // 预编译正则表达式
                    pattern = regexp.MustCompile("^" + regexStr + "$")
                    isRegex = true
                }
            }

            // 创建或获取参数节点
            if current.paramChild == nil {
                current.paramChild = &Node{
                    path:      segment,
                    children:  make(map[string]*Node),
                    isParam:   true,
                    paramName: paramName,
                }

                if isRegex {
                    current.paramChild.isRegex = true
                    current.paramChild.pattern = pattern
                }
            }

            current = current.paramChild
        } else {
            // 静态节点
            child, ok := current.children[segment]
            if !ok {
                child = &Node{
                    path:     segment,
                    children: make(map[string]*Node),
                }
                current.children[segment] = child
            }
            current = child
        }

        // 如果是最后一个段，设置处理函数
        if i == len(segments)-1 {
            current.handler = handler
        }
    }
}

// Find 在Radix Tree中查找匹配的处理函数（迭代实现）
func (n *Node) Find(path string, params map[string]string) (interface{}, bool) {
    // 处理根路径
    if path == "/" {
        return n.handler, n.handler != nil
    }

    // 标准化路径格式
    path = strings.Trim(path, "/")
    segments := strings.Split(path, "/")

    // 使用迭代而非递归进行查找
    current := n
    for i, segment := range segments {
        if segment == "" {
            continue
        }

        // 按优先级尝试匹配：Static > Regex > Param > Wildcard

        // 1. 静态匹配
        if child, ok := current.children[segment]; ok {
            current = child
            continue
        }

        // 2. 正则匹配
        if current.paramChild != nil && current.paramChild.isRegex {
            paramNode := current.paramChild
            if paramNode.pattern.MatchString(segment) {
                // 存储参数值
                params[paramNode.paramName] = segment
                current = paramNode
                continue
            }
        }

        // 3. 参数匹配
        if current.paramChild != nil && !current.paramChild.isRegex {
            paramNode := current.paramChild
            // 存储参数值
            params[paramNode.paramName] = segment
            current = paramNode
            continue
        }

        // 4. 通配符匹配
        if current.wildcardChild != nil {
            // 通配符匹配剩余所有路径
            remainingPath := strings.Join(segments[i:], "/")
            params["*"] = remainingPath
            return current.wildcardChild.handler, current.wildcardChild.handler != nil
        }

        // 没有匹配
        return nil, false
    }

    // 检查最终节点是否有处理函数
    return current.handler, current.handler != nil
}