package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fyerfyer/fyer-webframe/scaffold"
)

// registerTemplates 注册所有模板用于脚手架生成
// 这里我们仅使用已内嵌于scaffold包中的标准模板
func registerTemplates() []scaffold.Template {
	return scaffold.GetAllTemplates()
}

// ensureRequiredDirs 确保所有所需的目录已创建
func ensureRequiredDirs(projectPath string) error {
	// 获取所需目录列表
	dirs := scaffold.GetAllDirs()

	// 添加必要的基础目录
	baseDirs := []string{
		"controllers",
		"models",
		"views",
		"config",
		"public",
	}

	// 合并目录列表
	allDirs := append(baseDirs, dirs...)

	// 创建所有目录
	for _, dir := range allDirs {
		dirPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("无法创建目录 %s: %w", dirPath, err)
		}
	}

	return nil
}

// validateTemplate 验证模板配置的合法性
func validateTemplate(tmpl scaffold.Template) error {
	if tmpl.Path == "" {
		return fmt.Errorf("模板路径不能为空")
	}

	if tmpl.DestPath == "" {
		return fmt.Errorf("目标路径不能为空")
	}

	return nil
}

// validateTemplates 验证所有模板配置的合法性
func validateTemplates(templates []scaffold.Template) error {
	for _, tmpl := range templates {
		if err := validateTemplate(tmpl); err != nil {
			return fmt.Errorf("模板验证失败 [%s]: %w", tmpl.Path, err)
		}
	}
	return nil
}

// prepareTemplateData 准备模板数据
func prepareTemplateData(projectName string) scaffold.TemplateData {
	return scaffold.TemplateData{
		ProjectName: projectName,
		ModulePath:  "github.com/" + projectName,
	}
}

//// getVersion 获取当前框架版本
//func getVersion() string {
//	// 在真实环境中，这可能从框架包中导入版本信息
//	// 或通过构建时的注入获取
//	return "1.0.0"
//}