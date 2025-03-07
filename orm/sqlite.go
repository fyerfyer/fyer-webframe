package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strings"
)

type Sqlite struct {
	BaseDialect
}

func (s Sqlite) BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column) {
	if conflictCols == nil || len(conflictCols) == 0 {
		panic(errors.New("sqlite must have conflict columns"))
	}
	if len(cols) == 0 {
		panic(ferr.ErrUpsertRowNotFound)
	}

	builder.WriteString(" ON CONFLICT(")
	for index, col := range cols {
		col.model = s.model
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

// Quote SQLite使用双引号作为标识符引用符
func (s Sqlite) Quote(name string) string {
	return "\"" + name + "\""
}

// Placeholder SQLite使用问号作为占位符
func (s Sqlite) Placeholder(index int) string {
	return "?"
}

// Concat SQLite的字符串连接使用||运算符
func (s Sqlite) Concat(items ...string) string {
	builder := strings.Builder{}
	for i, item := range items {
		builder.WriteString(item)
		if i < len(items)-1 {
			builder.WriteString(" || ")
		}
	}
	return builder.String()
}

// IfNull SQLite使用IFNULL函数处理NULL值
func (s Sqlite) IfNull(expr string, defaultVal string) string {
	return "IFNULL(" + expr + ", " + defaultVal + ")"
}

// DateFormat SQLite的日期格式化函数
func (s Sqlite) DateFormat(dateExpr string, format string) string {
	return "strftime('" + format + "', " + dateExpr + ")"
}

// LimitOffset SQLite的LIMIT和OFFSET语法
//func (s Sqlite) LimitOffset(limit int, offset int) string {
//	if offset > 0 {
//		return "LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa(offset)
//	}
//	return "LIMIT " + strconv.Itoa(limit)
//}

// JulianDay 返回距离公元前4713年1月1日12时的天数
func (s Sqlite) JulianDay(dateExpr string) string {
	return "julianday(" + dateExpr + ")"
}

func init() {
	RegisterDialect("sqlite", &Sqlite{})
}