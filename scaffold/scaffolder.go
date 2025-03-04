package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ProjectScaffolder 是项目脚手架的核心结构体
type ProjectScaffolder struct {
	ProjectName string    // 项目名称
	ModulePath  string    // 模块路径
	OutputPath  string    // 输出路径
	CreatedAt   time.Time // 创建时间
}

// ScaffoldOption 定义脚手架选项函数
type ScaffoldOption func(*ProjectScaffolder)

// WithModulePath 设置自定义模块路径
func WithModulePath(modulePath string) ScaffoldOption {
	return func(s *ProjectScaffolder) {
		s.ModulePath = modulePath
	}
}

// WithOutputPath 设置自定义输出路径
func WithOutputPath(outputPath string) ScaffoldOption {
	return func(s *ProjectScaffolder) {
		s.OutputPath = outputPath
	}
}

// NewProjectScaffolder 创建一个新的项目脚手架实例
func NewProjectScaffolder(projectName string, opts ...ScaffoldOption) *ProjectScaffolder {
	// 创建默认的脚手架实例
	scaffolder := &ProjectScaffolder{
		ProjectName: projectName,
		ModulePath:  "github.com/" + projectName,
		OutputPath:  projectName,
		CreatedAt:   time.Now(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(scaffolder)
	}

	return scaffolder
}

// getFrameworkVersion 获取当前框架版本
//func getFrameworkVersion() string {
//	// 在实际生产环境中可以从框架package导入版本信息
//	// 这里简单返回一个固定版本
//	return "1.0.0"
//}

// Generate 生成项目脚手架
func (ps *ProjectScaffolder) Generate() error {
	// 1. 创建项目目录结构
	if err := ps.createProjectDirs(); err != nil {
		return fmt.Errorf("failed to create project directories: %w", err)
	}

	// 2. 生成项目文件
	if err := ps.generateProjectFiles(); err != nil {
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	// 3. 初始化Go模块
	if err := ps.initGoModule(); err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// 4. 安装依赖
	if err := ps.installDependencies(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	return nil
}

// createProjectDirs 创建项目目录结构
func (ps *ProjectScaffolder) createProjectDirs() error {
	baseDirs := []string{
		ps.OutputPath,
		filepath.Join(ps.OutputPath, "controllers"),
		filepath.Join(ps.OutputPath, "models"),
		filepath.Join(ps.OutputPath, "views"),
		filepath.Join(ps.OutputPath, "middleware"),
		filepath.Join(ps.OutputPath, "config"),
		filepath.Join(ps.OutputPath, "public"),
		filepath.Join(ps.OutputPath, "public", "css"),
		filepath.Join(ps.OutputPath, "public", "js"),
		filepath.Join(ps.OutputPath, "public", "images"),
	}

	for _, dir := range baseDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 添加所有在projectDirs中定义的目录
	for _, dir := range projectDirs {
		fullPath := filepath.Join(ps.OutputPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

// generateProjectFiles 生成项目文件
func (ps *ProjectScaffolder) generateProjectFiles() error {
	// 准备模板数据
	data := TemplateData{
		ProjectName: ps.ProjectName,
		ModulePath:  ps.ModulePath,
	}

	// 生成项目文件
	for _, tmpl := range GetAllTemplates() {
		if tmpl.IsDir {
			continue
		}

		// 读取模板内容
		content, err := GetTemplateContent(tmpl.Path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tmpl.Path, err)
		}

		// 解析模板内容
		parsedContent, err := ParseTemplateContent(content, data)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tmpl.Path, err)
		}

		// 确保目标目录存在
		destDir := filepath.Dir(filepath.Join(ps.OutputPath, tmpl.DestPath))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		// 写入文件
		destPath := filepath.Join(ps.OutputPath, tmpl.DestPath)
		if err := os.WriteFile(destPath, []byte(parsedContent), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}
	}

	return nil
}

// initGoModule 初始化Go模块
func (ps *ProjectScaffolder) initGoModule() error {
	// 检查是否已有go.mod文件
	if _, err := os.Stat(filepath.Join(ps.OutputPath, "go.mod")); err == nil {
		return nil // 已存在则不处理
	}

	// 执行go mod init
	cmd := exec.Command("go", "mod", "init", ps.ModulePath)
	cmd.Dir = ps.OutputPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// 添加框架依赖
	getCmd := exec.Command("go", "get", "github.com/fyerfyer/fyer-webframe")
	getCmd.Dir = ps.OutputPath
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	if err := getCmd.Run(); err != nil {
		return fmt.Errorf("failed to add framework dependency: %w", err)
	}

	// 整理依赖
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = ps.OutputPath
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy Go module: %w", err)
	}

	return nil
}

// installDependencies 安装项目依赖
func (ps *ProjectScaffolder) installDependencies() error {
	// 获取当前包的导入路径
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("failed to determine current package path")
	}

	// 获取框架根目录
	frameworkDir := filepath.Dir(filepath.Dir(currentFile))

	// 构建依赖映射
	deps := []string{
		// 添加框架依赖
		"github.com/fyerfyer/fyer-webframe",
	}

	// 如果是在开发模式，使用replace指令指向本地框架目录
	isDevMode := os.Getenv("WEBFRAME_DEV") == "1"
	if isDevMode {
		replaceCmd := exec.Command("go", "mod", "edit", "-replace",
			fmt.Sprintf("github.com/fyerfyer/fyer-webframe=%s", frameworkDir))
		replaceCmd.Dir = ps.OutputPath
		if err := replaceCmd.Run(); err != nil {
			return fmt.Errorf("failed to add module replacement: %w", err)
		}
	}

	// 安装依赖
	args := append([]string{"get"}, deps...)
	cmd := exec.Command("go", args...)
	cmd.Dir = ps.OutputPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// 整理go.mod
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = ps.OutputPath
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy Go module: %w", err)
	}

	return nil
}

// Run 运行生成的项目
func (ps *ProjectScaffolder) Run() error {
	fmt.Printf("Running project %s...\n", ps.ProjectName)

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = ps.OutputPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 注意这里使用Start而非Run，这样函数可以立即返回
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start project: %w", err)
	}

	// 给应用一点时间启动
	time.Sleep(500 * time.Millisecond)

	// 打印运行信息
	fmt.Printf("\n✅ Project %s is running at http://localhost:8080\n", ps.ProjectName)
	fmt.Println("Press Ctrl+C to stop the server")

	return nil
}

// GetProjectInfo 返回项目信息的字符串表示
func (ps *ProjectScaffolder) GetProjectInfo() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Project: %s\n", ps.ProjectName))
	sb.WriteString(fmt.Sprintf("Module: %s\n", ps.ModulePath))
	sb.WriteString(fmt.Sprintf("Created at: %s\n", ps.CreatedAt.Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("Location: %s\n", filepath.Join(ps.OutputPath)))

	return sb.String()
}
