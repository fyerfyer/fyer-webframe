package web

import (
	"bytes"
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

type TemplateOption func(*GoTemplate)

// WithPattern 设置模板文件匹配模式
func WithPattern(pattern string) TemplateOption {
	return func(t *GoTemplate) {
		t.tplPattern = pattern
	}
}

// WithFiles 设置模板文件列表
func WithFiles(files ...string) TemplateOption {
	return func(t *GoTemplate) {
		t.tplFiles = files
	}
}

func NewGoTemplate(opts ...TemplateOption) *GoTemplate {
	t := &GoTemplate{
		tpl: template.New(""),
	}

	for _, opt := range opts {
		opt(t)
	}

	// 如果设置了模板匹配模式，则立即加载模板
	if t.tplPattern != "" {
		if err := t.LoadFromGlob(t.tplPattern); err != nil {
			panic("load template error: " + err.Error())
		}
	}

	// 如果设置了模板文件列表，则立即加载模板
	if len(t.tplFiles) > 0 {
		if err := t.LoadFromFiles(t.tplFiles...); err != nil {
			panic("load template error: " + err.Error())
		}
	}

	return t
}

// LoadFromGlob 从匹配模式加载模板
func (g *GoTemplate) LoadFromGlob(pattern string) error {
	g.Lock()
	defer g.Unlock()

	var err error
	g.tpl, err = template.ParseGlob(pattern)
	if err != nil {
		return err
	}
	g.tplPattern = pattern
	return nil
}

// LoadFromFiles 从文件列表加载模板
func (g *GoTemplate) LoadFromFiles(files ...string) error {
	g.Lock()
	defer g.Unlock()

	var err error
	g.tpl, err = template.ParseFiles(files...)
	if err != nil {
		return err
	}
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

	buf := &bytes.Buffer{}
	err := g.tpl.ExecuteTemplate(buf, tplName, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
