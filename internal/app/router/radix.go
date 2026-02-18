package router

import "strings"

type nodeType int

const (
	nodeStatic nodeType = iota
	nodeParam
	nodeCatchAll
)

type radixNode struct {
	path     string
	children []*radixNode
	wildcard *radixNode
	nType    nodeType
	param    string
	handler  *Route
	indices  string
	priority uint32
}

func newRadixNode(path string, nType nodeType) *radixNode {
	return &radixNode{
		path:     path,
		nType:    nType,
		children: make([]*radixNode, 0),
	}
}

func (n *radixNode) insert(pattern string, route *Route) {
	if pattern == "" || pattern == "/" {
		n.handler = route
		n.priority++
		return
	}

	pattern = trimPrefixSlash(pattern)

	paramIdx := strings.Index(pattern, "{")
	if paramIdx == 0 {
		endIdx := strings.Index(pattern, "}")
		if endIdx == -1 {
			return
		}

		paramName := pattern[1:endIdx]
		rest := pattern[endIdx+1:]
		rest = trimPrefixSlash(rest)

		if n.wildcard == nil {
			nType := nodeParam
			if strings.HasPrefix(paramName, "...") {
				nType = nodeCatchAll
				paramName = paramName[3:]
			}
			n.wildcard = &radixNode{
				nType: nType,
				param: paramName,
			}
		}

		n.wildcard.priority++
		n.wildcard.insert(rest, route)
		return
	}

	if paramIdx > 0 {
		staticPart := pattern[:paramIdx]
		remaining := pattern[paramIdx:]

		for i := range n.children {
			child := n.children[i]
			if child.nType != nodeStatic {
				continue
			}

			commonPrefix := longestCommonPrefix(staticPart, child.path)
			if len(commonPrefix) == 0 {
				continue
			}

			if len(commonPrefix) < len(child.path) {
				n.splitChild(i, commonPrefix)
				child = n.children[i]
			}

			childRem := staticPart[len(commonPrefix):]
			childRem = trimPrefixSlash(childRem)
			if childRem == "" {
				child.priority++
				child.insert(remaining, route)
			} else {
				newChild := newRadixNode(childRem, nodeStatic)
				newChild.insert(remaining, route)
				child.addChild(newChild)
			}
			return
		}

		child := newRadixNode(staticPart, nodeStatic)
		child.insert(remaining, route)
		n.addChild(child)
		return
	}

	for i := range n.children {
		child := n.children[i]

		commonPrefix := longestCommonPrefix(pattern, child.path)
		if len(commonPrefix) == 0 {
			continue
		}

		if len(commonPrefix) < len(child.path) {
			n.splitChild(i, commonPrefix)
			child = n.children[i]
		}

		remaining := pattern[len(commonPrefix):]
		remaining = trimPrefixSlash(remaining)
		child.priority++
		child.insert(remaining, route)
		return
	}

	child := newRadixNode(pattern, nodeStatic)
	child.handler = route
	n.addChild(child)
}

func (n *radixNode) splitChild(index int, prefix string) {
	child := n.children[index]

	fullPath := child.path
	child.path = prefix

	remaining := fullPath[len(prefix):]

	newChild := newRadixNode(remaining, child.nType)
	newChild.handler = child.handler
	newChild.children = child.children
	newChild.wildcard = child.wildcard
	newChild.param = child.param

	child.handler = nil
	child.children = []*radixNode{newChild}
	child.wildcard = nil
	child.param = ""
}

func (n *radixNode) addChild(child *radixNode) {
	n.children = append(n.children, child)
	if len(child.path) > 0 {
		n.indices += string(child.path[0])
	}
}

func (n *radixNode) lookup(path string) (*Route, map[string]string) {
	params := make(map[string]string)
	result := n.search(path, params)
	return result, params
}

func (n *radixNode) search(path string, params map[string]string) *Route {
	path = trimPrefixSlash(path)

	if path == "" {
		return n.handler
	}

	for _, child := range n.children {
		if child.nType != nodeStatic || len(child.path) == 0 {
			continue
		}
		if len(path) < len(child.path) {
			continue
		}

		if path[:len(child.path)] != child.path {
			continue
		}

		remaining := path[len(child.path):]
		remaining = trimPrefixSlash(remaining)

		if result := child.search(remaining, params); result != nil {
			return result
		}
	}

	if n.wildcard != nil {
		if n.wildcard.nType == nodeCatchAll {
			params[n.wildcard.param] = path
			return n.wildcard.handler
		}

		slashIdx := strings.Index(path, "/")
		var paramValue, remaining string

		if slashIdx == -1 {
			paramValue = path
			remaining = ""
		} else {
			paramValue = path[:slashIdx]
			remaining = path[slashIdx+1:]
		}

		if paramValue != "" {
			params[n.wildcard.param] = paramValue
			return n.wildcard.search(remaining, params)
		}
	}

	return nil
}

func longestCommonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	i := 0
	for i < minLen && a[i] == b[i] {
		i++
	}

	return a[:i]
}

func trimPrefixSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s[1:]
	}
	return s
}
