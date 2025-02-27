package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"sync"
)

type Template interface {
	Render(ctx *Context, tplName string, data any) ([]byte, error)
	LoadFromGlob(pattern string) error
	LoadFromFiles(files ...string) error
	Reload() error
}

type GoTemplate struct {
	sync.RWMutex
	tplPattern string             // 模板文件匹配模式
	tplFiles   []string           // 模板文件列表
	tpl        *template.Template // 已编译的模板
}

type GoTemplateOption func(*GoTemplate)

// WithPattern 设置模板文件匹配模式
func WithPattern(pattern string) GoTemplateOption {
	return func(t *GoTemplate) {
		t.tplPattern = pattern
	}
}

// WithFiles 设置模板文件列表
func WithFiles(files ...string) GoTemplateOption {
	return func(t *GoTemplate) {
		t.tplFiles = files
	}
}

func NewGoTemplate(opts ...GoTemplateOption) *GoTemplate {
	t := &GoTemplate{
		tpl: template.New(""),
	}

	for _, opt := range opts {
		opt(t)
	}

	// 初始化时如果有模板，则尝试加载
	var err error
	if t.tplPattern != "" {
		err = t.LoadFromGlob(t.tplPattern)
	} else if len(t.tplFiles) > 0 {
		err = t.LoadFromFiles(t.tplFiles...)
	}

	// 如果加载失败，直接panic
	if err != nil {
		panic("load template error: " + err.Error())
	}

	return t
}

// LoadFromGlob 从匹配模式加载模板
func (g *GoTemplate) LoadFromGlob(pattern string) error {
	g.Lock()
	defer g.Unlock()

	temp, err := template.ParseGlob(pattern)
	if err != nil {
		return err
	}
	g.tpl = temp
	g.tplPattern = pattern
	return nil
}

// LoadFromFiles 从文件列表加载模板
func (g *GoTemplate) LoadFromFiles(files ...string) error {
	g.Lock()
	defer g.Unlock()

	temp, err := template.ParseFiles(files...)
	if err != nil {
		return err
	}
	g.tpl = temp
	g.tplFiles = files
	return nil
}

// Reload 重新加载模板
func (g *GoTemplate) Reload() error {
	if g.tplPattern != "" {
		return g.LoadFromGlob(g.tplPattern)
	}
	if len(g.tplFiles) > 0 {
		return g.LoadFromFiles(g.tplFiles...)
	}
	return nil
}

// Render 渲染模板
func (g *GoTemplate) Render(ctx *Context, tplName string, data any) ([]byte, error) {
	g.RLock()
	defer g.RUnlock()

	if g.tpl == nil {
		return nil, errors.New("template not initialized")
	}

	// 检查模板是否存在
	if g.tpl.Lookup(tplName) == nil {
		return nil, fmt.Errorf("template %s not found", tplName)
	}

	// 验证数据
	if data == nil {
		return nil, errors.New("template data cannot be nil")
	}

	buf := &bytes.Buffer{}
	err := g.tpl.ExecuteTemplate(buf, tplName, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
