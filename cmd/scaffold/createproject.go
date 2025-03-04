package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fyerfyer/fyer-webframe/scaffold"
)

// ProjectCreator 处理项目创建流程
type ProjectCreator struct {
	projectName string
	modulePath  string
	outputPath  string
	templates   []scaffold.Template
}

// NewProjectCreator 创建项目创建器
func NewProjectCreator(projectName string) (*ProjectCreator, error) {
	// 验证项目名称
	if err := validateProjectName(projectName); err != nil {
		return nil, err
	}

	// 准备默认的模块路径和输出路径
	modulePath := fmt.Sprintf("github.com/%s", projectName)
	outputPath := projectName

	return &ProjectCreator{
		projectName: projectName,
		modulePath:  modulePath,
		outputPath:  outputPath,
		templates:   registerTemplates(),
	}, nil
}

// SetModulePath 设置自定义模块路径
func (p *ProjectCreator) SetModulePath(modulePath string) {
	p.modulePath = modulePath
}

// SetOutputPath 设置自定义输出路径
func (p *ProjectCreator) SetOutputPath(outputPath string) {
	p.outputPath = outputPath
}

// Create 执行项目创建流程
func (p *ProjectCreator) Create() error {
	fmt.Printf("Creating project '%s'...\n", p.projectName)

	// 1. 检查目标目录是否存在
	if _, err := os.Stat(p.outputPath); err == nil {
		return fmt.Errorf("directory %s already exists", p.outputPath)
	}

	// 2. 创建项目目录结构
	if err := ensureRequiredDirs(p.outputPath); err != nil {
		return err
	}

	// 3. 验证模板
	if err := validateTemplates(p.templates); err != nil {
		return err
	}

	// 4. 准备模板数据
	data := prepareTemplateData(p.projectName)
	data.ModulePath = p.modulePath // 使用自定义模块路径

	// 5. 生成项目文件
	if err := p.generateFiles(data); err != nil {
		// 如果生成失败，尝试清理已创建的目录
		cleanUpOnFailure(p.outputPath)
		return err
	}

	// 6. 初始化 Git 仓库
	if err := initGitRepository(p.outputPath); err != nil {
		fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
	}

	// 7. 初始化 Go 模块
	if err := initGoModule(p.outputPath, p.modulePath); err != nil {
		return err
	}

	fmt.Printf("\n✅ Project '%s' created successfully!\n", p.projectName)
	fmt.Printf("Location: %s\n\n", filepath.Join(p.outputPath))
	fmt.Println("To run your new application:")
	fmt.Printf("  cd %s\n", p.projectName)
	fmt.Println("  go run .")

	return nil
}

// generateFiles 生成所有项目文件
func (p *ProjectCreator) generateFiles(data scaffold.TemplateData) error {
	fmt.Println("Generating project files...")

	for _, tmpl := range p.templates {
		// 跳过处理go.mod文件，现在由命令行工具生成
		if tmpl.DestPath == "go.mod" {
			continue
		}

		if tmpl.IsDir {
			continue
		}

		// 读取模板内容
		content, err := scaffold.GetTemplateContent(tmpl.Path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmpl.Path, err)
		}

		// 解析模板内容
		parsedContent, err := scaffold.ParseTemplateContent(content, data)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmpl.Path, err)
		}

		// 确保目标目录存在
		destPath := filepath.Join(p.outputPath, tmpl.DestPath)
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		// 写入文件
		if err := os.WriteFile(destPath, []byte(parsedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		fmt.Printf("  Created: %s\n", tmpl.DestPath)
	}

	return nil
}

// validateProjectName 验证项目名称
func validateProjectName(name string) error {
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

	// 检查是否是Go保留关键字
	goKeywords := []string{
		"break", "default", "func", "interface", "select", "case", "defer",
		"go", "map", "struct", "chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type", "continue", "for",
		"import", "return", "var",
	}

	for _, keyword := range goKeywords {
		if name == keyword {
			return fmt.Errorf("project name cannot be a Go keyword: %s", name)
		}
	}

	return nil
}

// initGitRepository 初始化Git仓库
func initGitRepository(path string) error {
	// 如果git命令不存在，则跳过此步骤
	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	fmt.Println("Initializing git repository...")

	// 创建.gitignore文件
	gitignore := []byte(`# IDE files
.idea/
.vscode/
*.iml

# Output directories
/bin/
/dist/
/vendor/

# Dependency directories
/node_modules/

# Debug files
*.log
*.out

# Go specific
*.test
*.prof
`)

	gitignorePath := filepath.Join(path, ".gitignore")
	if err := os.WriteFile(gitignorePath, gitignore, 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}

// initGoModule 初始化Go模块
func initGoModule(path string, modulePath string) error {
	fmt.Println("Initializing Go module...")

	// 直接初始化Go模块
	initCmd := exec.Command("go", "mod", "init", modulePath)
	initCmd.Dir = path
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// 添加框架依赖
	getCmd := exec.Command("go", "get", fmt.Sprintf("github.com/fyerfyer/fyer-webframe"))
	getCmd.Dir = path
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	if err := getCmd.Run(); err != nil {
		return fmt.Errorf("failed to add framework dependency: %w", err)
	}

	// 如果是开发模式，添加replace指令
	isDevMode := os.Getenv("WEBFRAME_DEV") == "1"
	if isDevMode {
		_, currentFile, _, ok := runtime.Caller(0)
		if ok {
			frameworkDir := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
			replaceCmd := exec.Command("go", "mod", "edit", "-replace",
				fmt.Sprintf("github.com/fyerfyer/fyer-webframe=%s", frameworkDir))
			replaceCmd.Dir = path
			replaceCmd.Stdout = os.Stdout
			replaceCmd.Stderr = os.Stderr
			if err := replaceCmd.Run(); err != nil {
				return fmt.Errorf("failed to add module replacement: %w", err)
			}
		}
	}

	// 整理依赖
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = path
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy dependencies: %w", err)
	}

	return nil
}

// cleanUpOnFailure 在失败时清理已创建的目录
func cleanUpOnFailure(path string) {
	fmt.Printf("Cleaning up %s...\n", path)
	if err := os.RemoveAll(path); err != nil {
		fmt.Printf("Warning: Failed to clean up directory %s: %v\n", path, err)
	}
}

// RunProject 运行生成的项目
func RunProject(projectPath string) error {
	fmt.Printf("Starting project in %s...\n", projectPath)

	// 使用脚手架库提供的运行功能
	scaffolder := scaffold.NewProjectScaffolder("", scaffold.WithOutputPath(projectPath))
	if err := scaffolder.Run(); err != nil {
		return err
	}

	fmt.Println("Project is running. Press Ctrl+C to stop.")
	return nil
}

// ProjectInfo 包含项目信息
type ProjectInfo struct {
	Name      string
	Path      string
	Created   time.Time
	ModPath   string
	Framework string
}

// FormatProjectInfo 格式化项目信息为可读形式
func FormatProjectInfo(info ProjectInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project: %s\n", info.Name))
	sb.WriteString(fmt.Sprintf("Location: %s\n", info.Path))
	sb.WriteString(fmt.Sprintf("Created: %s\n", info.Created.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Module: %s\n", info.ModPath))
	sb.WriteString(fmt.Sprintf("Framework: fyerfyer/fyer-webframe v%s\n", info.Framework))
	return sb.String()
}
