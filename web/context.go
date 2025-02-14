package web

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Context struct {
	Req   *http.Request
	Resp  http.ResponseWriter
	Param map[string]string
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
	if err := c.Req.ParseForm(); err != nil {
		return StringValue{
			Error: err,
		}
	}

	val, ok := c.Req.Form[key]
	if !ok {
		return StringValue{
			Error: errors.New("key not found"),
		}
	}

	return StringValue{
		Value: val[0],
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
