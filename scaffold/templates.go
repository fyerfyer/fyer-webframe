package scaffold

import (
	"embed"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

// Template 表示一个项目模板文件
type Template struct {
	Path     string // 模板在FS中的路径
	DestPath string // 目标路径（相对于项目根目录）
	IsDir    bool   // 是否为目录
}

// 项目基本结构模板定义
var projectTemplates = []Template{
	{Path: "templates/main.tmpl", DestPath: "main.go", IsDir: false},
	{Path: "templates/config.tmpl", DestPath: "config/config.go", IsDir: false},
	{Path: "templates/controllers/home.tmpl", DestPath: "controllers/home.go", IsDir: false},
	{Path: "templates/models/user.tmpl", DestPath: "models/user.go", IsDir: false},
	{Path: "templates/views/home.tmpl", DestPath: "views/home.html", IsDir: false},
	{Path: "templates/views/layout.tmpl", DestPath: "views/layout.html", IsDir: false},
}

// 需要创建的空目录
var projectDirs = []string{
	"middlewares",
	"public/css",
	"public/js",
	"public/images",
	"config",
}

// TemplateData 包含生成项目需要的数据
type TemplateData struct {
	ProjectName string // 项目名称
	ModulePath  string // Go模块路径
	Title       string // 页面标题
	Message     string // 页面消息
	CurrentYear string // 当前年份
}

// ParseTemplateContent 解析模板内容
func ParseTemplateContent(content string, data TemplateData) (string, error) {
	// 设置默认值
	if data.Title == "" {
		data.Title = data.ProjectName
	}

	if data.CurrentYear == "" {
		data.CurrentYear = time.Now().Format("2006")
	}

	// 确保模板引擎能找到所有需要的变量
	if data.Message == "" {
		data.Message = "Welcome to " + data.ProjectName
	}

	// 检查是否是HTML模板文件
	if strings.Contains(content, "{{define") || strings.Contains(content, "{{block") {
		// 简单替换项目名称等信息，而不破坏HTML模板语法
		content = strings.ReplaceAll(content, "{{ .ProjectName }}", data.ProjectName)
		content = strings.ReplaceAll(content, "{{ .ModulePath }}", data.ModulePath)
		content = strings.ReplaceAll(content, "{{ .CurrentYear }}", data.CurrentYear)
		content = strings.ReplaceAll(content, "{{ .Title }}", data.Title)
		content = strings.ReplaceAll(content, "{{ .Message }}", data.Message)
		return content, nil
	}

	// 常规模板处理
	tmpl, err := template.New("template").Parse(content)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// GetTemplateContent 从嵌入式FS中读取模板内容
func GetTemplateContent(path string) (string, error) {
	content, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetAllTemplates 返回所有项目模板
func GetAllTemplates() []Template {
	return projectTemplates
}

// GetAllDirs 返回所有需要创建的目录
func GetAllDirs() []string {
	return projectDirs
}
