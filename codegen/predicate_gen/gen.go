package predicate_gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Field struct {
	Name string
	Type string
}

type ImportInfo struct {
	Path  string // 完整导入路径
	Alias string // 别名（如果有）
}

type StructInfo struct {
	Name    string
	Fields  []Field
	Pkg     string
	Imports map[string]ImportInfo
}

func Generate(inputFile string, outputDir string) error {
	// 解析Go源文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse file error: %w", err)
	}

	// 创建导入包映射
	importMap := make(map[string]ImportInfo)

	// 收集所有导入包信息
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		parts := strings.Split(importPath, "/")
		defaultPkgName := parts[len(parts)-1]

		var info ImportInfo
		info.Path = importPath

		if imp.Name != nil {
			// 如果有别名，使用别名作为键，并记录别名
			info.Alias = imp.Name.Name
			importMap[imp.Name.Name] = info
		} else {
			// 没有别名，使用包名最后一部分作为键
			importMap[defaultPkgName] = info
		}
	}

	// 收集所有结构体信息
	var structs []StructInfo
	pkg := node.Name.Name

	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := t.Type.(*ast.StructType); ok {
				info := StructInfo{
					Name:    t.Name.Name,
					Pkg:     pkg,
					Imports: make(map[string]ImportInfo),
				}

				for _, field := range structType.Fields.List {
					if !ast.IsExported(field.Names[0].Name) {
						continue
					}

					typeStr, pkgName := extractTypeInfo(field.Type)

					// 如果字段类型使用了外部包，添加到导入列表
					if pkgName != "" {
						if importInfo, exists := importMap[pkgName]; exists {
							info.Imports[pkgName] = importInfo
						} else {
							// 如果包名不在导入列表中，说明包没有被使用，源文件有语法错误
							panic(fmt.Sprintf("package %s not found in imports", pkgName))
						}
					}

					info.Fields = append(info.Fields, Field{
						Name: field.Names[0].Name,
						Type: typeStr,
					})
				}
				structs = append(structs, info)
			}
		}
		return true
	})

	// 生成代码
	for _, st := range structs {
		if err := generateForStruct(st, outputDir); err != nil {
			return fmt.Errorf("generate code error: %w", err)
		}
	}

	return nil
}

// 修改 extractTypeInfo 函数，移除特殊处理
func extractTypeInfo(expr ast.Expr) (typeStr string, pkgName string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, ""
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name, ident.Name
		}
		return fmt.Sprintf("%v.%s", t.X, t.Sel.Name), ""
	case *ast.StarExpr:
		str, pkgName := extractTypeInfo(t.X)
		return "*" + str, pkgName
	case *ast.ArrayType:
		str, pkgName := extractTypeInfo(t.Elt)
		return "[]" + str, pkgName
	default:
		return fmt.Sprintf("%T", expr), ""
	}
}

func typeToString(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, false
	case *ast.SelectorExpr:
		// 检查是否是 sql 包的类型
		if ident, ok := t.X.(*ast.Ident); ok && ident.Name == "sql" {
			return "sql." + t.Sel.Name, true
		}
		typeString, _ := typeToString(t.X)
		return typeString + "." + t.Sel.Name, false
	case *ast.StarExpr:
		str, hasSql := typeToString(t.X)
		return "*" + str, hasSql
	case *ast.ArrayType:
		str, hasSql := typeToString(t.Elt)
		return "[]" + str, hasSql
	default:
		return fmt.Sprintf("%T", expr), false
	}
}

func generateForStruct(info StructInfo, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 修改文件名生成逻辑
	fileName := strings.ToLower(info.Name) + ".gen.go"
	filePath := filepath.Join(outputDir, fileName)

	// 创建输出文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 解析模板
	tmpl, err := template.New("predicate").Parse(predicateTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(file, info)
}
