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

type StructInfo struct {
	Name      string
	Fields    []Field
	Pkg       string
	HasSqlPkg bool // 新增字段，标记是否需要导入 database/sql
}

func Generate(inputFile string, outputDir string) error {
	// 解析Go源文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse file error: %w", err)
	}

	// 收集所有结构体信息
	var structs []StructInfo
	pkg := node.Name.Name

	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := t.Type.(*ast.StructType); ok {
				info := StructInfo{
					Name: t.Name.Name,
					Pkg:  pkg,
				}

				for _, field := range structType.Fields.List {
					// 只处理导出的字段
					if !ast.IsExported(field.Names[0].Name) {
						continue
					}

					typeStr, hasSql := typeToString(field.Type)
					if hasSql {
						info.HasSqlPkg = true
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

	// 生成代码
	return tmpl.Execute(file, info)
}
