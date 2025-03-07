package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strconv"
	"strings"
)

type Postgresql struct {
	BaseDialect
}

func (p Postgresql) BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column) {
	if conflictCols == nil || len(conflictCols) == 0 {
		panic(errors.New("postgresql must have conflict columns"))
	}
	if len(cols) == 0 {
		panic(ferr.ErrUpsertRowNotFound)
	}

	builder.WriteString(" ON CONFLICT(")
	for index, col := range cols {
		col.model = p.model
		col.Build(builder)
		if index != len(cols)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString(") DO UPDATE SET ")

	for index, col := range cols {
		col.BuildWithoutQuote(builder)
		builder.WriteString(" = EXCLUDED.")
		col.BuildWithoutQuote(builder)
		if index != len(cols)-1 {
			builder.WriteString(", ")
		}
	}
}

// Quote PostgreSQL使用双引号作为标识符引用符
func (p Postgresql) Quote(name string) string {
	return "\"" + name + "\""
}

// Placeholder PostgreSQL使用$n作为参数占位符
func (p Postgresql) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

// Concat PostgreSQL的字符串连接使用||运算符
func (p Postgresql) Concat(items ...string) string {
	builder := strings.Builder{}
	for i, item := range items {
		builder.WriteString(item)
		if i < len(items)-1 {
			builder.WriteString(" || ")
		}
	}
	return builder.String()
}

// IfNull PostgreSQL使用COALESCE函数处理NULL值
func (p Postgresql) IfNull(expr string, defaultVal string) string {
	return "COALESCE(" + expr + ", " + defaultVal + ")"
}

// DateFormat PostgreSQL的日期格式化函数
func (p Postgresql) DateFormat(dateExpr string, format string) string {
	return "TO_CHAR(" + dateExpr + ", '" + format + "')"
}

// LimitOffset PostgreSQL的LIMIT和OFFSET语法
//func (p Postgresql) LimitOffset(limit int, offset int) string {
//	if limit > 0 && offset < 0 {
//		return "LIMIT " + strconv.Itoa(limit)
//	} else if limit < 0 && offset > 0 {
//		return "OFFSET " + strconv.Itoa(offset)
//	} else if limit > 0 && offset > 0 {
//		return "LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
//	}
//
//	return ""
//}

// JsonExtract PostgreSQL的JSON提取操作
func (p Postgresql) JsonExtract(jsonExpr string, path string) string {
	return jsonExpr + "->" + path
}

// JsonExtractText PostgreSQL的JSON文本提取操作
func (p Postgresql) JsonExtractText(jsonExpr string, path string) string {
	return jsonExpr + "->>" + path
}

func init() {
	RegisterDialect("postgresql", &Postgresql{})
}