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

    // 参数子节点映射，键为参数名
    paramChildren map[string]*Node

    // 正则参数子节点，例如 :id([0-9]+)
    regexChildren []*Node

    // 通配符子节点，例如 *
    wildcardChild *Node

    // 处理函数
    handler interface{}

    // 是否是参数节点
    isParam bool
    
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
        paramChildren: make(map[string]*Node),  // 初始化参数子节点映射
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

    wildcardCount := 0
    wildcardIndex := -1
    for i, segment := range segments {
        if segment == "*" {
            wildcardCount++
            wildcardIndex = i
        }
    }

    // 只允许一个通配符段
    if wildcardCount > 1 {
        panic("only one wildcard segment is allowed in path")
    }

    if wildcardIndex >= 0 && wildcardIndex < len(segments)-1 {
        panic("wildcard segment must be the last segment in path")
    }

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
                    paramChildren: make(map[string]*Node),
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
            if strings.Contains(paramName, "(") {
                regexStart := strings.Index(paramName, "(")

                if !strings.Contains(paramName, ")") {
                    panic(fmt.Sprintf("invalid regex pattern in '%s': missing closing parenthesis", segment))
                }

                regexEnd := strings.LastIndex(paramName, ")")

                if regexEnd <= regexStart {
                    panic(fmt.Sprintf("invalid regex pattern in '%s': misplaced parentheses", segment))
                }

                regexStr := paramName[regexStart + 1:regexEnd]
                paramName = paramName[:regexStart]

                // 检查是否有相同参数名的正则节点
                for _, regexNode := range current.regexChildren {
                    if regexNode.paramName == paramName && regexNode.pattern.String() != "^"+regexStr+"$" {
                        panic(fmt.Sprintf("conflicting parameter name '%s' with different regex patterns", paramName))
                    }
                }

                var err error
                pattern, err = regexp.Compile("^" + regexStr + "$")
                if err != nil {
                    panic(fmt.Sprintf("invalid regex pattern: %s - %s", regexStr, err))
                }
                isRegex = true
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
                        paramChildren: make(map[string]*Node),
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
                // 普通参数节点 - 使用参数名称作为键
                
                // 检查是否有参数路径冲突（终止节点的情况）
                // 如果尝试注册的是终止节点，并且已存在其他参数节点也是终止节点
                // 这种情况应该被视为冲突，需要 panic

                // todo: 其实这样检测忽略了参数路径位于中间的路径冲突，这样的检测是不完整的，不过用户不应该这样使用路由
                if i == len(segments) - 1 {
                    // 检查所有现有的参数节点
                    for existingParamName, existingParamNode := range current.paramChildren {
                        // 如果参数名不同，且已有节点是终止节点（有处理函数）
                        if existingParamName != paramName && existingParamNode.handler != nil {
                            panic(fmt.Sprintf("conflicting parameter names at same position: '%s' and '%s'", 
                                existingParamName, paramName))
                        }
                    }
                }
                
                if _, exists := current.paramChildren[paramName]; !exists {
                    // 如果该参数名称不存在，创建新节点
                    current.paramChildren[paramName] = &Node{
                        path: segment,
                        children: make(map[string]*Node),
                        paramChildren: make(map[string]*Node),
                        regexChildren: make([]*Node, 0),
                        isParam: true,
                        paramName: paramName,
                    }
                } else if i == len(segments) - 1 && current.paramChildren[paramName].handler != nil &&
                          len(current.paramChildren[paramName].children) == 0 {
                    // 只在没有子节点的情况下不允许重复注册
                    panic(fmt.Sprintf("duplicate router '%s' registered", path))
                }
                // 移动到对应参数名的节点
                current = current.paramChildren[paramName]
            }
        } else {
            // 静态节点
            child, ok := current.children[segment]
            if !ok {
                child = &Node{
                    path: segment,
                    children: make(map[string]*Node),
                    paramChildren: make(map[string]*Node),
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

    //fmt.Printf("[DEBUG] Finding path in RadixTree: %s\n", path)

    // 使用迭代而非递归进行查找
    current := n
    for i := 0; i < len(segments); {
        segment := segments[i]
        //fmt.Printf("[DEBUG] Processing segment: '%s' at index %d\n", segment, i)
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
        // 尝试所有可能的参数匹配，先检查当前节点路径下是否有可以继续匹配的
        paramMatched := false
        if len(current.paramChildren) > 0 {
            for paramName, paramNode := range current.paramChildren {
                // 检查此参数路径是否能匹配后续段
                canMatchLater := true

                if i < len(segments)-1 {
                    // 还有更多段需要匹配
                    nextSegment := segments[i+1]

                    // 检查参数节点的子节点是否能匹配下一段
                    nextSegmentCanMatch := false

                    // 检查静态子节点
                    if _, ok := paramNode.children[nextSegment]; ok {
                        nextSegmentCanMatch = true
                    }

                    // 检查正则子节点
                    if !nextSegmentCanMatch {
                        for _, regexChild := range paramNode.regexChildren {
                            if regexChild.pattern.MatchString(nextSegment) {
                                nextSegmentCanMatch = true
                                break
                            }
                        }
                    }

                    // 检查参数子节点
                    if !nextSegmentCanMatch && len(paramNode.paramChildren) > 0 {
                        nextSegmentCanMatch = true
                    }

                    // 检查通配符子节点
                    if !nextSegmentCanMatch && paramNode.wildcardChild != nil {
                        nextSegmentCanMatch = true
                    }

                    canMatchLater = nextSegmentCanMatch
                }

                if canMatchLater {
                    // 这个参数节点可以匹配当前段并且可能能匹配后续段
                    params[paramName] = segment
                    //fmt.Printf("[DEBUG] Matched parameter: %s = %s\n", paramName, segment)
                    current = paramNode
                    i++
                    paramMatched = true
                    break
                }
            }

            // 如果找到匹配的参数路径，继续下一轮循环
            if paramMatched {
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
        //fmt.Printf("[DEBUG] Final matched params: %v\n", params)
        return current.handler, true
    }

    // 如果当前节点无处理函数但有通配符子节点，返回通配符子节点的处理函数
    if current.wildcardChild != nil {
        params["*"] = ""
        //fmt.Printf("[DEBUG] Final matched params: %v\n", params)
        return current.wildcardChild.handler, current.wildcardChild.handler != nil
    }

    // 没有匹配的处理函数
    return nil, false
}