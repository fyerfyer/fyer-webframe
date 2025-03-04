package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fyerfyer/fyer-webframe/scaffold"
)

var (
	// 命令行参数
	projectName = flag.String("name", "", "Project name (required)")
	modulePath  = flag.String("module", "", "Go module path (default: github.com/{project-name})")
	outputPath  = flag.String("output", "", "Output directory (default: ./{project-name})")
	runFlag     = flag.Bool("run", false, "Run the project after creation")
)

// usage 显示使用帮助信息
func usage() {
	fmt.Printf("Fyer Web Framework Project Scaffold\n\n")
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Printf("  %s -name myproject\n", os.Args[0])
	fmt.Printf("  %s -name myproject -module example.com/myproject\n", os.Args[0])
	fmt.Printf("  %s -name myproject -output ./projects/myproject\n", os.Args[0])
	fmt.Printf("  %s -name myproject -run\n", os.Args[0])
}

func main() {
	flag.Usage = usage
	flag.Parse()

	// 验证必须的项目名称参数
	if *projectName == "" {
		fmt.Println("Error: Project name is required")
		flag.Usage()
		os.Exit(1)
	}

	// 验证项目名是否合法
	if err := scaffold.ValidateProjectName(*projectName); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// 设置默认的模块路径和输出路径
	modPath := *modulePath
	if modPath == "" {
		modPath = "github.com/" + *projectName
	}

	outPath := *outputPath
	if outPath == "" {
		outPath = *projectName
	}

	// 清理输出路径
	outPath = filepath.Clean(outPath)

	fmt.Printf("Creating new project '%s'...\n\n", *projectName)

	// 创建项目
	creator, err := NewProjectCreator(*projectName)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	if modPath != "" {
		creator.SetModulePath(modPath)
	}

	if outPath != "" {
		creator.SetOutputPath(outPath)
	}

	startTime := time.Now()

	// 执行项目创建
	if err := creator.Create(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)

	// 显示项目信息
	showProjectInfo(*projectName, outPath, modPath, duration)

	// 如果设置了运行标志，则运行项目
	if *runFlag {
		fmt.Printf("\nRunning project %s...\n", *projectName)
		if err := RunProject(outPath); err != nil {
			fmt.Printf("Error running project: %s\n", err)
			os.Exit(1)
		}
	}
}

// showProjectInfo 显示项目创建信息
func showProjectInfo(name, path, module string, duration time.Duration) {
	fmt.Println("\n✅ Project created successfully!")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Project name:      %s\n", name)
	fmt.Printf("Location:          %s\n", path)
	fmt.Printf("Go module:         %s\n", module)
	fmt.Printf("Creation time:     %v\n", duration.Round(time.Millisecond))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("\nTo run your new project:")
	fmt.Printf("  cd %s\n", path)
	fmt.Println("  go run .")
	fmt.Println()
	fmt.Printf("Happy coding with Fyer Web Framework!\n\n")
}