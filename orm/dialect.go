package orm

import (
	"strings"
)

type Dialect interface {
	// BuildUpsert 构建 UPSERT 语句
	BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column)

	// Quote 根据数据库方言对标识符(表名、列名等)进行引用
	Quote(name string) string

	// Placeholder 生成参数占位符
	Placeholder(index int) string

	// Concat 字符串连接函数
	Concat(items ...string) string

	// IfNull 处理空值
	IfNull(expr string, defaultVal string) string
}

var (
	dialects = make(map[string]Dialect)
)

func RegisterDialect(name string, dialect Dialect) {
	dialects[name] = dialect
}

func Get(name string) Dialect {
	return dialects[name]
}

type BaseDialect struct {
	model *model
}

func (b *BaseDialect) setModel(m *model) {
	b.model = m
}

// 提供默认实现，可被具体方言覆盖
func (b *BaseDialect) Quote(name string) string {
	return "`" + name + "`"
}

// 默认使用问号作为占位符
func (b *BaseDialect) Placeholder(index int) string {
	return "?"
}

// 默认的字符串连接实现
func (b *BaseDialect) Concat(items ...string) string {
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

// 默认的空值处理实现
func (b *BaseDialect) IfNull(expr string, defaultVal string) string {
	return "IFNULL(" + expr + ", " + defaultVal + ")"
}