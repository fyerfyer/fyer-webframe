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
	c.Resp.Header().Set("Content-Type", "application/json")
	c.Resp.WriteHeader(code)

	encoder := json.NewEncoder(c.Resp)
	return encoder.Encode(v)
}
