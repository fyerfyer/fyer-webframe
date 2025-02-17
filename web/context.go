package web

import (
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
	tplEngine      Template // 新增模板引擎字段
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

	c.RespStatusCode = code
	c.RespData = data
	c.unhandled = true
	return nil
}

// RespString 返回字符串响应
func (c *Context) RespString(code int, str string) error {
	c.RespStatusCode = code
	c.RespData = []byte(str)
	c.unhandled = true
	return nil
}

// RespBytes 返回字节数组响应
func (c *Context) RespBytes(code int, data []byte) error {
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

	data, err := c.tplEngine.Render(c, tplName, data)
	if err != nil {
		return err
	}

	c.RespData = data.([]byte)
	c.unhandled = true
	return nil
}
