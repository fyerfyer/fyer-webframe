package web

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"path/filepath"
)

// ContentType 常用的内容类型常量
const (
	ContentTypeJSON           = "application/json; charset=utf-8"
	ContentTypeXML            = "application/xml; charset=utf-8"
	ContentTypePlain          = "text/plain; charset=utf-8"
	ContentTypeHTML           = "text/html; charset=utf-8"
	ContentTypeForm           = "application/x-www-form-urlencoded"
	ContentTypeMultipartForm  = "multipart/form-data"
	ContentTypeOctetStream    = "application/octet-stream"
	ContentTypeAttachment     = "attachment"
	ContentTypeEventStream    = "text/event-stream; charset=utf-8"
	ContentTypeYAML           = "application/yaml; charset=utf-8"
	ContentTypeProblemJSON    = "application/problem+json"
	ContentTypeProblemXML     = "application/problem+xml"
)

// ResponseHelper 为 Context 添加响应帮助方法
type ResponseHelper interface {
	// JSON 返回 JSON 格式的响应
	JSON(code int, data any) error

	// XML 返回 XML 格式的响应
	XML(code int, data any) error

	// String 返回纯文本响应
	String(code int, format string, values ...any) error

	// HTML 返回 HTML 响应
	HTML(code int, html string) error

	// Attachment 发送文件作为附件
	Attachment(path, name string) error

	// File 返回文件内容
	File(filepath string) error

	// FileFromFS 从文件系统返回文件
	FileFromFS(filepath string, fs http.FileSystem) error

	// Template 渲染模板
	Template(name string, data any) error

	// Created 返回 201 Created 响应
	Created(uri string, data any) error

	// NoContent 返回 204 No Content 响应
	NoContent() error

	// BadRequest 返回 400 Bad Request 响应
	BadRequest(message string) error

	// Unauthorized 返回 401 Unauthorized 响应
	Unauthorized(message string) error

	// Forbidden 返回 403 Forbidden 响应
	Forbidden(message string) error

	// NotFound 返回 404 Not Found 响应
	NotFound(message string) error

	// InternalServerError 返回 500 Internal Server Error 响应
	InternalServerError(message string) error

	// Redirect 重定向到指定的 URL
	Redirect(code int, url string) error

	// StreamEvent 发送服务器发送事件 (SSE)
	StreamEvent(event string, data any) error

	// Problem 返回 RFC7807 问题详情
	Problem(code int, problem *ProblemDetails) error
}

// ProblemDetails RFC7807 问题详情
type ProblemDetails struct {
	Type     string `json:"type,omitempty" xml:"type,omitempty"`
	Title    string `json:"title" xml:"title"`
	Status   int    `json:"status" xml:"status"`
	Detail   string `json:"detail,omitempty" xml:"detail,omitempty"`
	Instance string `json:"instance,omitempty" xml:"instance,omitempty"`
}

// 以下是 Context 添加的响应方法实现

// JSON 返回 JSON 格式的响应
func (c *Context) JSON(code int, data any) error {
	c.Resp.Header().Set("Content-Type", ContentTypeJSON)
	c.RespStatusCode = code
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.RespData = jsonData
	c.unhandled = true
	return nil
}

// XML 返回 XML 格式的响应
func (c *Context) XML(code int, data any) error {
	c.Resp.Header().Set("Content-Type", ContentTypeXML)
	c.RespStatusCode = code
	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	c.RespData = xmlData
	c.unhandled = true
	return nil
}

// String 返回纯文本响应
func (c *Context) String(code int, format string, values ...any) error {
	c.Resp.Header().Set("Content-Type", ContentTypePlain)
	c.RespStatusCode = code
	c.RespData = []byte(fmt.Sprintf(format, values...))
	c.unhandled = true
	return nil
}

// HTML 返回 HTML 响应
func (c *Context) HTML(code int, html string) error {
	c.Resp.Header().Set("Content-Type", ContentTypeHTML)
	c.RespStatusCode = code
	c.RespData = []byte(html)
	c.unhandled = true
	return nil
}

// Template 渲染模板并返回
func (c *Context) Template(name string, data any) error {
	if c.tplEngine == nil {
		return fmt.Errorf("template engine not configured")
	}

	result, err := c.tplEngine.Render(c, name, data)
	if err != nil {
		return err
	}

	c.Resp.Header().Set("Content-Type", ContentTypeHTML)
	c.RespStatusCode = http.StatusOK
	c.RespData = result
	c.unhandled = true
	return nil
}

// Attachment 下载附件
func (c *Context) Attachment(path, name string) error {
	if name == "" {
		name = filepath.Base(path)
	}
	c.Resp.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", ContentTypeAttachment, name))
	return c.File(path)
}

// File 返回文件内容
func (c *Context) File(filepath string) error {
	http.ServeFile(c.Resp, c.Req, filepath)
	c.unhandled = false
	return nil
}

// FileFromFS 从文件系统返回文件
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) error {
	http.FileServer(fs).ServeHTTP(c.Resp, c.Req)
	c.unhandled = false
	return nil
}

// Created 返回 201 Created 响应
func (c *Context) Created(uri string, data any) error {
	if uri != "" {
		c.Resp.Header().Set("Location", uri)
	}
	return c.JSON(http.StatusCreated, data)
}

// NoContent 返回 204 No Content 响应
func (c *Context) NoContent() error {
	c.RespStatusCode = http.StatusNoContent
	c.unhandled = true
	return nil
}

// BadRequest 返回 400 Bad Request 响应
func (c *Context) BadRequest(message string) error {
	return c.JSON(http.StatusBadRequest, map[string]string{"error": message})
}

// Unauthorized 返回 401 Unauthorized 响应
func (c *Context) Unauthorized(message string) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{"error": message})
}

// Forbidden 返回 403 Forbidden 响应
func (c *Context) Forbidden(message string) error {
	return c.JSON(http.StatusForbidden, map[string]string{"error": message})
}

// NotFound 返回 404 Not Found 响应
func (c *Context) NotFound(message string) error {
	if message == "" {
		message = "resource not found"
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": message})
}

// InternalServerError 返回 500 Internal Server Error 响应
func (c *Context) InternalServerError(message string) error {
	if message == "" {
		message = "internal server error"
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": message})
}

// Redirect 重定向到指定的 URL
func (c *Context) Redirect(code int, url string) error {
	http.Redirect(c.Resp, c.Req, url, code)
	c.unhandled = false
	return nil
}

// StreamEvent 发送服务器发送事件 (SSE)
func (c *Context) StreamEvent(event string, data any) error {
	c.Resp.Header().Set("Content-Type", ContentTypeEventStream)
	c.Resp.Header().Set("Cache-Control", "no-cache")
	c.Resp.Header().Set("Connection", "keep-alive")
	c.unhandled = false

	// 格式化数据
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		dataStr = string(jsonData)
	}

	if event != "" {
		fmt.Fprintf(c.Resp, "event: %s\n", event)
	}
	fmt.Fprintf(c.Resp, "data: %s\n\n", dataStr)

	if flusher, ok := c.Resp.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// Problem 返回 RFC7807 问题详情
func (c *Context) Problem(code int, problem *ProblemDetails) error {
	if problem == nil {
		return fmt.Errorf("problem details cannot be nil")
	}

	// 设置状态码
	problem.Status = code

	// 根据请求的 Accept 头部选择响应格式
	accept := c.Req.Header.Get("Accept")
	if accept == ContentTypeProblemXML {
		c.Resp.Header().Set("Content-Type", ContentTypeProblemXML)
		xmlData, err := xml.MarshalIndent(problem, "", "  ")
		if err != nil {
			return err
		}
		c.RespStatusCode = code
		c.RespData = xmlData
		c.unhandled = true
		return nil
	}

	// 默认数据类型为JSON
	c.Resp.Header().Set("Content-Type", ContentTypeProblemJSON)
	jsonData, err := json.Marshal(problem)
	if err != nil {
		return err
	}
	c.RespStatusCode = code
	c.RespData = jsonData
	c.unhandled = true
	return nil
}

