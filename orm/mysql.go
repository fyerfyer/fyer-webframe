package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strings"
)

type Mysql struct {
	BaseDialect
}

func (m Mysql) BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column) {
	if conflictCols != nil {
		panic(errors.New("mysql does not support conflict columns"))
	}
	m.buildUpsert(builder, cols)
}

func (m Mysql) buildUpsert(builder *strings.Builder, cols []*Column) {
	if len(cols) == 0 {
		panic(ferr.ErrUpsertRowNotFound)
	}

	builder.WriteString(" ON DUPLICATE KEY UPDATE ")

	for index, col := range cols {
		if index > 0 {
			builder.WriteString(", ")
		}

		// 注入模型信息
		col.model = m.model
		col.Build(builder)
		builder.WriteString(" = VALUES(")
		col.BuildWithoutQuote(builder)
		builder.WriteByte(')')
	}
}

// Quote 使用反引号作为MySQL的标识符引用符
func (m Mysql) Quote(name string) string {
	return "`" + name + "`"
}

// Placeholder MySQL使用问号作为占位符
func (m Mysql) Placeholder(index int) string {
	return "?"
}

// Concat MySQL的字符串连接函数
func (m Mysql) Concat(items ...string) string {
	builder := strings.Builder{}
	builder.WriteString("CONCAT(")
	for i, item := range items {
		builder.WriteString(item)
		if i < len(items)-1 {
			builder.WriteString(", ")
		}
	}
	builder.WriteString(")")
	return builder.String()
}

// IfNull MySQL的IFNULL函数
func (m Mysql) IfNull(expr string, defaultVal string) string {
	return "IFNULL(" + expr + ", " + defaultVal + ")"
}

// DateFormat MySQL日期格式化函数
func (m Mysql) DateFormat(dateExpr string, format string) string {
	return "DATE_FORMAT(" + dateExpr + ", '" + format + "')"
}

// LimitOffset MySQL的LIMIT和OFFSET语法
//func (m Mysql) LimitOffset(limit int, offset int) string {
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

func init() {
	RegisterDialect("mysql", &Mysql{})
}