package web

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestStaticRoute(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}
	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/"},
		{method: http.MethodGet, path: "/user"},
		{method: http.MethodGet, path: "/admin/login"},
		{method: http.MethodGet, path: "/user/login"},
		{method: http.MethodGet, path: "/user/logout"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	wantRouter := Router{
		routerTrees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"login": &node{
								path:    "login",
								handler: mockHandlerFunc,
							},
							"logout": &node{
								path:    "logout",
								handler: mockHandlerFunc,
							},
						},
						handler: mockHandlerFunc,
					},
					"admin": &node{
						path: "admin",
						children: map[string]*node{
							"login": &node{
								path:    "login",
								handler: mockHandlerFunc,
							},
						},
					},
				},
				handler: mockHandlerFunc,
			},
		},
	}

	msg, ok := nodeEqual(r.routerTrees[http.MethodGet], wantRouter.routerTrees[http.MethodGet])
	assert.True(t, ok, msg)

	// 异常测试
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "", mockHandlerFunc)
	}, "path cannot be empty")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/", mockHandlerFunc)
	}, "path must not end with /")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user", mockHandlerFunc)
	}, "cannot register the same path")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user//abc", mockHandlerFunc)
	}, "path cannot contain //")
}

func TestWildcardRoute(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}
	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/*"},
		{method: http.MethodGet, path: "/user/login/*"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	wantRouter := Router{
		routerTrees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"login": &node{
								path: "login",
								children: map[string]*node{
									"*": &node{
										path:         "*",
										hasStarParam: true,
										handler:      mockHandlerFunc,
									},
								},
							},
							"*": &node{
								path:         "*",
								hasStarParam: true,
								handler:      mockHandlerFunc,
							},
						},
					},
				},
			},
		},
	}

	msg, ok := nodeEqual(r.routerTrees[http.MethodGet], wantRouter.routerTrees[http.MethodGet])
	assert.True(t, ok, msg)

	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/*", mockHandlerFunc)
	}, "cannot register the same path")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/*/a/*", mockHandlerFunc)
	}, "should not support more than one wildcard")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/*/abc", mockHandlerFunc)
	}, "should not have path after wildcard")
}

func TestParamRoute(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}
	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/:id"},
		{method: http.MethodGet, path: "/user/:id/login"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	wantRouter := Router{
		routerTrees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"id": &node{
								path: "id",
								children: map[string]*node{
									"login": &node{
										path:    "login",
										handler: mockHandlerFunc,
									},
								},
								isParam: true,
								handler: mockHandlerFunc,
							},
						},
						hasParamChild: true,
					},
				},
			},
		},
	}

	msg, ok := nodeEqual(r.routerTrees[http.MethodGet], wantRouter.routerTrees[http.MethodGet])
	assert.True(t, ok, msg)

	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/*", mockHandlerFunc)
	}, "cannot register * and / in the same path")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/:name", mockHandlerFunc)
	}, "cannot register the same param path")
	assert.NotPanics(t, func() {
		r.addHandler(http.MethodGet, "/user/:id/logout", mockHandlerFunc)
	}, "should support more than one param in different path")
}

func TestStaticRouteFound(t *testing.T) {
	r := NewRouter()
	testCtx := &Context{}
	mockHandlerFunc := func(ctx *Context) {}
	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/"},
		{method: http.MethodGet, path: "/user"},
		{method: http.MethodGet, path: "/admin/login"},
		{method: http.MethodGet, path: "/user/login"},
		{method: http.MethodGet, path: "/user/logout"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	n, ok := r.findHandler(http.MethodGet, "/", testCtx)
	assert.True(t, ok, "root path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc).Pointer(), reflect.ValueOf(n.handler).Pointer(), "root handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/user", testCtx)
	assert.True(t, ok, "user path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc).Pointer(), reflect.ValueOf(n.handler).Pointer(), "user handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/admin/login", testCtx)
	assert.True(t, ok, "admin login path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc).Pointer(), reflect.ValueOf(n.handler).Pointer(), "admin login handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/login", testCtx)
	assert.True(t, ok, "user login path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc).Pointer(), reflect.ValueOf(n.handler).Pointer(), "user login handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/logout", testCtx)
	assert.True(t, ok, "user logout path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc).Pointer(), reflect.ValueOf(n.handler).Pointer(), "user logout handler not equal")
}

func TestWildcardRouteFound(t *testing.T) {
	r := NewRouter()
	testCtx := &Context{}
	mockHandlerFunc1 := func(ctx *Context) {}
	mockHandlerFunc2 := func(ctx *Context) {}

	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/*"},
		{method: http.MethodGet, path: "/user/a"},
		{method: http.MethodGet, path: "/user/a/b"},
	}

	for _, router := range testRoutes {
		if strings.Contains(router.path, "*") {
			r.addHandler(router.method, router.path, mockHandlerFunc1)
		} else {
			r.addHandler(router.method, router.path, mockHandlerFunc2)
		}
	}

	n, ok := r.findHandler(http.MethodGet, "/user/a", testCtx)
	assert.True(t, ok, "/user/a path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc2).Pointer(),
		reflect.ValueOf(n.handler).Pointer(),
		"/user/a handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/a/b", testCtx)
	assert.True(t, ok, "user login path not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc2).Pointer(),
		reflect.ValueOf(n.handler).Pointer(),
		"/user/a/b handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/abc/d", testCtx)
	assert.True(t, ok, "wildcard not found")
	assert.Equal(t, reflect.ValueOf(mockHandlerFunc1).Pointer(),
		reflect.ValueOf(n.handler).Pointer(),
		"/user/abc/d handler not equal")

	n, ok = r.findHandler(http.MethodGet, "/users/abc/d/e", testCtx)
	assert.False(t, ok, "should not found unregister path")
}

func TestParamFound(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}

	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/:id"},
		{method: http.MethodGet, path: "/user/:name/login"},
		{method: http.MethodGet, path: "/admin/:name/logout"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	n, ok := r.findHandler(http.MethodGet, "/user/123", &Context{})
	assert.True(t, ok, "/user/:id path not found")
	id, ok := n.Param["id"]
	assert.True(t, ok, "param id not found")
	assert.Equal(t, id, "123", "param id not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/abc/login", &Context{})
	assert.True(t, ok, "/user/:name/login path not found")
	name, ok := n.Param["name"]
	assert.True(t, ok, "param name not found")
	assert.Equal(t, name, "abc", "param name not equal")

	_, ok = r.findHandler(http.MethodGet, "/admin/abc", &Context{})
	assert.False(t, ok, "should not match /admin/:name/logout")
}

func TestRegexRoute(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}
	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/:id([0-9]+)"},
		{method: http.MethodGet, path: "/user/:name([a-z]+)/profile"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	wantRouter := Router{
		routerTrees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"id": &node{
								path:         "id",
								isParam:      true,
								isRegex:      true,
								regexPattern: regexp.MustCompile("[0-9]+"),
								handler:      mockHandlerFunc,
							},
							"name": &node{
								path:         "name",
								isParam:      true,
								isRegex:      true,
								regexPattern: regexp.MustCompile("[a-z]+"),
								children: map[string]*node{
									"profile": &node{
										path:    "profile",
										handler: mockHandlerFunc,
									},
								},
							},
						},
						hasParamChild: true,
					},
				},
			},
		},
	}

	msg, ok := nodeEqual(r.routerTrees[http.MethodGet], wantRouter.routerTrees[http.MethodGet])
	assert.True(t, ok, msg)

	// 异常测试
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/:id([0-9]+)", mockHandlerFunc)
	}, "cannot register the same regex path")
	assert.Panics(t, func() {
		r.addHandler(http.MethodGet, "/user/:id([invalid)", mockHandlerFunc)
	}, "should panic with invalid regex pattern")
}

func TestRegexRouteFound(t *testing.T) {
	r := NewRouter()
	mockHandlerFunc := func(ctx *Context) {}

	testRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/user/:id([0-9]+)"},
		{method: http.MethodGet, path: "/user/:name([a-z]+)/profile"},
	}

	for _, router := range testRoutes {
		r.addHandler(router.method, router.path, mockHandlerFunc)
	}

	n, ok := r.findHandler(http.MethodGet, "/user/123", &Context{})
	assert.True(t, ok, "/user/:id([0-9]+) path not found")
	id, ok := n.Param["id"]
	assert.True(t, ok, "param id not found")
	assert.Equal(t, "123", id, "param id not equal")

	n, ok = r.findHandler(http.MethodGet, "/user/abc/profile", &Context{})
	assert.True(t, ok, "/user/:name([a-z]+)/profile path not found")
	name, ok := n.Param["name"]
	assert.True(t, ok, "param name not found")
	assert.Equal(t, "abc", name, "param name not equal")

	_, ok = r.findHandler(http.MethodGet, "/user/abc", &Context{})
	assert.False(t, ok, "should not match non-numeric id")

	_, ok = r.findHandler(http.MethodGet, "/user/123/profile", &Context{})
	assert.False(t, ok, "should not match numeric name")
}

func nodeEqual(a, b *node) (string, bool) {
	if a.path != b.path {
		return fmt.Sprint("path are not equal: a:",
			a.path,
			" ",
			"b:",
			b.path), false
	}

	if a.hasParamChild != b.hasParamChild {
		return fmt.Sprint("hasParamChild not equal, anode: ",
			a.hasParamChild,
			" ",
			"bnode: ",
			b.hasParamChild), false
	}

	//if a.hasStarChild != b.hasStarChild {
	//	return fmt.Sprint("hasStarChild not equal, anode: ",
	//		a.path,
	//		" ",
	//		a.hasStarChild,
	//		" ",
	//		"bnode: ",
	//		b.path,
	//		" ",
	//		b.hasStarChild), false
	//}

	if a.isRegex != b.isRegex {
		return fmt.Sprint("isRegex not equal, anode: ",
			a.isRegex,
			" ",
			"bnode: ",
			b.isRegex), false
	}

	if a.regexPattern != nil && b.regexPattern != nil {
		if a.regexPattern.String() != b.regexPattern.String() {
			return fmt.Sprint("regexPattern not equal, anode: ",
				a.regexPattern.String(),
				" ",
				"bnode: ",
				b.regexPattern.String()), false
		}
	}

	if a.Param != nil && b.Param != nil {
		if len(a.Param) != len(b.Param) {
			return fmt.Sprint("param numbers are not equal"), false
		}

		for key, value := range a.Param {
			bValue, ok := b.Param[key]
			if !ok {
				return fmt.Sprint("param not found: ", key, "in the second node"), false
			}

			if value != bValue {
				return fmt.Sprint("param not equal: ", key, "anode: ", value, " ", "bnode: ", bValue), false
			}
		}
	}

	if a.children != nil && b.children != nil {
		if len(a.children) != len(b.children) {
			return fmt.Sprint("child numbers are not equal"), false
		}

		if reflect.ValueOf(a.handler).Pointer() != reflect.ValueOf(b.handler).Pointer() {
			return fmt.Sprint("handler are not equal"), false
		}

		for key, child := range a.children {
			bChild, ok := b.children[key]
			if !ok {
				return fmt.Sprint("child not found: ", key, "in the second node"), false
			}

			if msg, ok := nodeEqual(child, bChild); !ok {
				return fmt.Sprint("child not equal: ", msg), false
			}

			if child.hasStarParam != bChild.hasStarParam {
				return fmt.Sprint("hasStarParam not equal, anode: ",
					a.hasStarParam,
					" ",
					"bnode: ",
					b.hasStarParam), false
			}

			if child.isParam != bChild.isParam {
				return fmt.Sprint("isParam not equal, anode: ",
					a.isParam,
					" ",
					"bnode: ",
					b.isParam), false
			}
		}
	}

	return "", true
}
