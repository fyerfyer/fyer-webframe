package router

import (
    "fmt"
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

    // 正则参数子节点，例如 :id([0-9]+)
    regexChildren []*Node 

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
        regexChildren: make([]*Node, 0),
    }
}

// Insert 将路径和对应的处理函数插入到Radix Tree中
func (n *Node) Insert(path string, handler interface{}) {
    // 处理根路径特殊情况
    if path == "/" {
        if n.handler != nil {
            panic(fmt.Sprintf("duplicate route '%s' registered", path))
        }
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
                    path: "*",
                    children: make(map[string]*Node),
                    regexChildren: make([]*Node, 0),
                }
            } else if i == len(segments) - 1 && current.wildcardChild.handler != nil {
                panic(fmt.Sprintf("duplicate router '%s' registered", path))
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
                    regexStr := paramName[regexStart + 1:regexEnd]
                    paramName = paramName[:regexStart]

                    // 预编译正则表达式
                    pattern = regexp.MustCompile("^" + regexStr + "$")
                    isRegex = true
                }
            }

            // 创建或获取参数节点
            if isRegex {
                // 正则参数节点
                var matchingNode *Node

                // 检查是否已存在相同模式的正则节点
                for _, regexNode := range current.regexChildren {
                    if regexNode.paramName == paramName && regexNode.pattern.String() == pattern.String() {
                        matchingNode = regexNode
                        break
                    }
                }

                if matchingNode == nil {
                    // 如果没有相同模式的节点，创建新节点
                    matchingNode = &Node{
                        path: segment,
                        children: make(map[string]*Node),
                        regexChildren: make([]*Node, 0),
                        isParam: true,
                        isRegex: true,
                        paramName: paramName,
                        pattern: pattern,
                    }
                    current.regexChildren = append(current.regexChildren, matchingNode)
                } else if i == len(segments) - 1 && matchingNode.handler != nil {
                    panic(fmt.Sprintf("duplicate router '%s' registered", path))
                }
                current = matchingNode
            } else {
                // 普通参数节点
                if current.paramChild == nil {
                    current.paramChild = &Node{
                        path: segment,
                        children: make(map[string]*Node),
                        regexChildren: make([]*Node, 0),
                        isParam: true,
                        paramName: paramName,
                    }
                } else if i == len(segments) - 1 && current.paramChild.handler != nil && current.paramChild.paramName == paramName {
                    panic(fmt.Sprintf("duplicate router '%s' registered", path))
                }
                current = current.paramChild
            }
        } else {
            // 静态节点
            child, ok := current.children[segment]
            if !ok {
                child = &Node{
                    path: segment,
                    children: make(map[string]*Node),
                    regexChildren: make([]*Node, 0),
                }
                current.children[segment] = child
            } else if i == len(segments) - 1 && child.handler != nil {
                panic(fmt.Sprintf("duplicate router '%s' registered", path))
            }
            current = child
        }

        // 如果是最后一个段，设置处理函数
        if i == len(segments) - 1 {
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
    for i := 0; i < len(segments); {
        segment := segments[i]
        if segment == "" {
            i++
            continue
        }

        // 1. 静态匹配 (最高优先级)
        if child, ok := current.children[segment]; ok {
            current = child
            i++
            continue
        }

        // 2. 正则匹配 (次高优先级)
        regexMatched := false
        for _, regexChild := range current.regexChildren {
            if regexChild.pattern.MatchString(segment) {
                params[regexChild.paramName] = segment
                current = regexChild
                i++
                regexMatched = true
                break
            }
        }
        if regexMatched {
            continue
        }

        // 3. 参数匹配 (第三优先级)
        if current.paramChild != nil {
            // 正常参数匹配
            params[current.paramChild.paramName] = segment
            
            // 这里是关键：如果参数子节点下还有子节点并且当前不是最后一个段，
            // 我们需要继续检查之后的段能否匹配参数子节点的子节点
            if i < len(segments)-1 {
                // 检查参数子节点是否有子节点可以匹配后续段
                nextSegment := segments[i+1]
                // 如果有静态子节点、参数子节点或通配符子节点可以匹配下一段，继续匹配
                paramChildCanMatch := false
                if _, ok := current.paramChild.children[nextSegment]; ok {
                    paramChildCanMatch = true
                } else {
                    // 检查正则子节点
                    for _, regexChild := range current.paramChild.regexChildren {
                        if regexChild.pattern.MatchString(nextSegment) {
                            paramChildCanMatch = true
                            break
                        }
                    }
                    
                    if !paramChildCanMatch && current.paramChild.paramChild != nil {
                        paramChildCanMatch = true
                    } else if !paramChildCanMatch && current.paramChild.wildcardChild != nil {
                        paramChildCanMatch = true
                    }
                }
                
                if paramChildCanMatch {
                    current = current.paramChild
                    i++
                    continue
                }
            } else {
                // 如果是最后一个段，直接使用参数匹配
                current = current.paramChild
                i++
                continue
            }
        }

        // 4. 通配符匹配 (最低优先级)
        if current.wildcardChild != nil {
            // 通配符匹配剩余所有路径
            remainingPath := strings.Join(segments[i:], "/")
            params["*"] = remainingPath
            return current.wildcardChild.handler, current.wildcardChild.handler != nil
        }

        // 没有匹配
        return nil, false
    }

    // 完成所有段的匹配后，检查是否有处理函数
    if current.handler != nil {
        return current.handler, true
    }

    // 如果当前节点无处理函数但有通配符子节点，返回通配符子节点的处理函数
    if current.wildcardChild != nil {
        params["*"] = ""
        return current.wildcardChild.handler, current.wildcardChild.handler != nil
    }

    // 没有匹配的处理函数
    return nil, false
}