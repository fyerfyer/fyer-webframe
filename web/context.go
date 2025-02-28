package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type Context struct {
	Req            *http.Request
	Resp           http.ResponseWriter
	Param          map[string]string
	RouteURL       string
	RespStatusCode int
	RespData       []byte
	unhandled      bool
	tplEngine      Template
	UserValues     map[string]any
	Context        context.Context
	aborted        bool
}

func (c *Context) Abort() {
	c.aborted = true
}

func (c *Context) IsAborted() bool {
	return c.aborted
}

func (c *Context) Next(next HandlerFunc) {
	if !c.aborted {
		next(c)
	}
}

func (c *Context) BindJSON(v any) error {
	if c.Req.Body == nil {
		return errors.New("missing request body")
	}

	decoder := json.NewDecoder(c.Req.Body)
	return decoder.Decode(v)
}

type StringValue struct {
	Value string
	Error error
}

func (c *Context) FormValue(key string) StringValue {
	val := c.Req.FormValue(key)
	if val == "" {
		return StringValue{
			Error: errors.New("key not found"),
		}
	}

	return StringValue{
		Value: val,
	}
}

func (c *Context) PathParam(key string) StringValue {
	val, ok := c.Param[key]
	if !ok {
		return StringValue{
			Error: errors.New("key not found"),
		}
	}

	return StringValue{
		Value: val,
	}
}

func (c *Context) JSON(code int, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.Resp.WriteHeader(code)
	_, err = c.Resp.Write(data)
	return err
}

// RespJSON 返回JSON响应
func (c *Context) RespJSON(code int, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// 设置content-type头
	c.Resp.Header().Set("Content-Type", "application/json; charset=utf-8")

	c.RespStatusCode = code
	c.RespData = data
	c.unhandled = true
	return nil
}

// RespString 返回字符串响应
func (c *Context) RespString(code int, str string) error {
	// 设置content-type头
	c.Resp.Header().Set("Content-Type", "text/plain; charset=utf-8")

	c.RespStatusCode = code
	c.RespData = []byte(str)
	c.unhandled = true
	return nil
}

// RespBytes 返回字节数组响应
func (c *Context) RespBytes(code int, data []byte) error {
	// 设置content-type头
	c.Resp.Header().Set("Content-Type", "application/octet-stream")

	c.RespStatusCode = code
	c.RespData = data
	c.unhandled = true
	return nil
}

// Render 渲染模板
func (c *Context) Render(tplName string, data any) error {
	if c.tplEngine == nil {
		return errors.New("template engine not found")
	}

	result, err := c.tplEngine.Render(c, tplName, data)
	if err != nil {
		return err
	}

	// 设置content-type头
	c.Resp.Header().Set("Content-Type", "text/html; charset=utf-8")

	c.RespData = result
	c.unhandled = true
	return nil
}

// Redirect 执行重定向操作
func (c *Context) Redirect(code int, url string) error {
	c.Resp.Header().Set("Location", url)
	c.RespStatusCode = code
	c.unhandled = false // 已经设置好响应数据了
	return nil
}

// SetHeader 设置请求头
func (c *Context) SetHeader(key, value string) *Context {
	c.Resp.Header().Set(key, value)
	return c
}

// Status 设置HTTP状态码
func (c *Context) Status(code int) *Context {
	c.RespStatusCode = code
	return c
}
