package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
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
	funcMap    template.FuncMap   // 自定义模板函数
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

// WithFuncMap 设置自定义模板函数
func WithFuncMap(funcMap template.FuncMap) GoTemplateOption {
	return func(t *GoTemplate) {
		t.funcMap = funcMap
	}
}

func NewGoTemplate(opts ...GoTemplateOption) *GoTemplate {
	t := &GoTemplate{
		tpl:     template.New(""),
		funcMap: make(template.FuncMap),
	}

	for _, opt := range opts {
		opt(t)
	}

	// 初始化模板函数
	t.tpl = t.tpl.Funcs(t.funcMap)

	// 初始化时如果有模板，则尝试加载
	var err error
	if t.tplPattern != "" {
		err = t.LoadFromGlob(t.tplPattern)
	} else if len(t.tplFiles) > 0 {
		err = t.LoadFromFiles(t.tplFiles...)
	}

	// 如果加载失败，记录错误但不panic
	if err != nil {
		fmt.Printf("Warning: Failed to load templates: %v\n", err)
	}

	return t
}

// LoadFromGlob 从匹配模式加载模板
func (g *GoTemplate) LoadFromGlob(pattern string) error {
	g.Lock()
	defer g.Unlock()

	// 先获取所有匹配的文件
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no files match pattern %s", pattern)
	}

	// 创建新模板
	temp := template.New(filepath.Base(matches[0])).Funcs(g.funcMap)
	temp, err = temp.ParseGlob(pattern)
	if err != nil {
		return fmt.Errorf("failed to parse glob: %w", err)
	}

	// 记录模板信息
	g.tpl = temp
	g.tplPattern = pattern
	g.tplFiles = matches
	return nil
}

// LoadFromFiles 从文件列表加载模板
func (g *GoTemplate) LoadFromFiles(files ...string) error {
	g.Lock()
	defer g.Unlock()

	if len(files) == 0 {
		return errors.New("no template files provided")
	}

	// 验证所有文件都存在
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("template file %s does not exist", file)
		}
	}

	// 创建新模板
	temp := template.New(filepath.Base(files[0])).Funcs(g.funcMap)
	temp, err := temp.ParseFiles(files...)
	if err != nil {
		return fmt.Errorf("failed to parse files: %w", err)
	}

	// 记录模板信息
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
	return errors.New("no template source defined")
}

// DebugTemplateNames 返回所有已加载模板的名称，用于调试
//func (g *GoTemplate) DebugTemplateNames() []string {
//	g.RLock()
//	defer g.RUnlock()
//
//	if g.tpl == nil {
//		return nil
//	}
//
//	var names []string
//	for _, t := range g.tpl.Templates() {
//		names = append(names, t.Name())
//	}
//	return names
//}

// Render 渲染模板
func (g *GoTemplate) Render(ctx *Context, tplName string, data any) ([]byte, error) {
	g.RLock()
	defer g.RUnlock()

	//fmt.Printf("DEBUG Render: Starting render for template '%s'\n", tplName)
	if g.tpl == nil {
		//fmt.Println("DEBUG Render: Template object is nil")
		return nil, errors.New("template not initialized")
	}

	buf := &bytes.Buffer{}

	// Debug: 打印所有可用模板
	//templateNames := g.DebugTemplateNames()
	//fmt.Printf("DEBUG Render: Available templates: %v\n", templateNames)

	// 检查模板是否存在
	tmpl := g.tpl.Lookup(tplName)
	if tmpl == nil {
		//fmt.Printf("DEBUG Render: Template '%s' not found\n", tplName)
		return nil, fmt.Errorf("template %s not found", tplName)
	}
	//fmt.Printf("DEBUG Render: Template '%s' found\n", tplName)

	// 验证数据
	if data == nil {
		//fmt.Println("DEBUG Render: Template data is nil")
		return nil, errors.New("template data cannot be nil")
	}

	//fmt.Printf("DEBUG Render: Executing template '%s'\n", tplName)

	// 使用ExecuteTemplate确保正确处理嵌套模板
	err := g.tpl.ExecuteTemplate(buf, tplName, data)
	if err != nil {
		//fmt.Printf("DEBUG Render: Template execution error: %v\n", err)
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.Bytes()
	//fmt.Printf("DEBUG Render: Template execution successful. Generated %d bytes\n", len(result))
	//fmt.Printf("DEBUG Render: Template result begins with: %s\n", string(result[:min(len(result), 100)]))

	return result, nil
}

// LoadFromFS 从文件系统加载模板
func (g *GoTemplate) LoadFromFS(fsys fs.FS, patterns ...string) error {
	g.Lock()
	defer g.Unlock()

	if len(patterns) == 0 {
		return errors.New("no patterns provided")
	}

	// 创建新模板
	temp := template.New("").Funcs(g.funcMap)
	temp, err := temp.ParseFS(fsys, patterns...)
	if err != nil {
		return fmt.Errorf("failed to parse fs: %w", err)
	}

	// 记录模板信息
	g.tpl = temp
	return nil
}
