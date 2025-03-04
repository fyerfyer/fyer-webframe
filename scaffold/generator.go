package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ProjectGenerator 负责生成项目文件和目录结构
type ProjectGenerator struct {
	ProjectName string    // 项目名称
	ModulePath  string    // Go模块路径
	OutputPath  string    // 输出路径
	Templates   []Template // 项目模板
}

// GeneratorOption 定义生成器选项函数
type GeneratorOption func(*ProjectGenerator)

// WithGenModulePath 设置自定义模块路径
func WithGenModulePath(modulePath string) GeneratorOption {
	return func(g *ProjectGenerator) {
		g.ModulePath = modulePath
	}
}

// WithGenOutputPath 设置自定义输出路径
func WithGenOutputPath(outputPath string) GeneratorOption {
	return func(g *ProjectGenerator) {
		g.OutputPath = outputPath
	}
}

// WithGenVersion 设置框架版本
//func WithGenVersion(version string) GeneratorOption {
//	return func(g *ProjectGenerator) {
//		g.Version = version
//	}
//}

// WithGenTemplates 设置自定义模板列表
func WithGenTemplates(templates []Template) GeneratorOption {
	return func(g *ProjectGenerator) {
		g.Templates = templates
	}
}

// NewProjectGenerator 创建一个新的项目生成器
func NewProjectGenerator(projectName string, opts ...GeneratorOption) *ProjectGenerator {
	// 创建默认的项目生成器
	generator := &ProjectGenerator{
		ProjectName: projectName,
		ModulePath:  "github.com/" + projectName,
		OutputPath:  projectName,
		Templates:   projectTemplates,
	}

	// 应用选项
	for _, opt := range opts {
		opt(generator)
	}

	return generator
}

// Generate 生成项目
func (g *ProjectGenerator) Generate() error {
	// 1. 创建项目目录结构
	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// 2. 生成项目文件
	if err := g.generateFiles(); err != nil {
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	return nil
}

// createDirectoryStructure 创建项目目录结构
func (g *ProjectGenerator) createDirectoryStructure() error {
	// 首先创建基本路径
	if err := os.MkdirAll(g.OutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// 创建所有模板文件的父目录
	for _, tmpl := range g.Templates {
		if tmpl.IsDir {
			dirPath := filepath.Join(g.OutputPath, tmpl.DestPath)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
			}
		} else {
			dirPath := filepath.Dir(filepath.Join(g.OutputPath, tmpl.DestPath))
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
			}
		}
	}

	// 创建其他必要的空目录
	for _, dir := range projectDirs {
		dirPath := filepath.Join(g.OutputPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	return nil
}

// generateFiles 生成项目文件
func (g *ProjectGenerator) generateFiles() error {
	// 准备模板数据
	data := TemplateData{
		ProjectName: g.ProjectName,
		ModulePath:  g.ModulePath,
	}

	// 为每个模板生成文件
	for _, tmpl := range g.Templates {
		if tmpl.IsDir {
			continue
		}

		// 获取模板内容
		content, err := GetTemplateContent(tmpl.Path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmpl.Path, err)
		}

		// 解析模板内容
		parsed, err := g.parseTemplate(tmpl.Path, content, data)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmpl.Path, err)
		}

		// 写入文件
		destPath := filepath.Join(g.OutputPath, tmpl.DestPath)
		if err := os.WriteFile(destPath, []byte(parsed), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
	}

	return nil
}

// parseTemplate 解析模板内容
func (g *ProjectGenerator) parseTemplate(name, content string, data interface{}) (string, error) {
	tmpl, err := template.New(filepath.Base(name)).Parse(content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ValidateProjectName 验证项目名是否有效
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// 检查是否包含无效字符
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	for _, char := range name {
		if !strings.ContainsRune(validChars, char) {
			return fmt.Errorf("project name contains invalid characters: only alphanumeric, underscore and dash are allowed")
		}
	}

	// 检查是否是Go关键字
	keywords := []string{"break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var"}

	for _, keyword := range keywords {
		if name == keyword {
			return fmt.Errorf("project name cannot be Go keyword: %s", name)
		}
	}

	return nil
}