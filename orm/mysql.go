package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"reflect"
	"strconv"
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

// CreateTableSQL 为MySQL生成建表语句
func (m Mysql) CreateTableSQL(model *model) string {
	// 先调用基本实现生成通用的SQL
	baseSQL := m.BaseDialect.CreateTableSQL(model)

	// 添加MySQL特有的表选项
	return baseSQL + " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;";
}

// AlterTableSQL 实现MySQL特定的表结构修改语句
func (m Mysql) AlterTableSQL(model *model, existingTable *model) string {
	// 调用基本实现
	return m.BaseDialect.AlterTableSQL(model, existingTable)
}

// TableExistsSQL 实现MySQL检查表是否存在的SQL
func (m Mysql) TableExistsSQL(schema, table string) string {
	if schema == "" {
		return "SELECT 1 FROM information_schema.tables WHERE table_name = '" + table + "'"
	}
	return "SELECT 1 FROM information_schema.tables WHERE table_schema = '" + schema + "' AND table_name = '" + table + "'"
}

// ColumnType 为MySQL实现Go类型到SQL类型的映射
func (m Mysql) ColumnType(f *field) string {
	// 如果字段明确指定了SQL类型，直接使用
	if f.sqlType != "" {
		return f.sqlType
	}

	// 根据Go类型映射MySQL类型
	switch f.typ.Kind() {
	case reflect.Bool:
		return "TINYINT(1)"
	case reflect.Int, reflect.Int32:
		if f.autoIncr {
			return "INT AUTO_INCREMENT"
		}
		return "INT"
	case reflect.Int8:
		return "TINYINT"
	case reflect.Int16:
		return "SMALLINT"
	case reflect.Int64:
		if f.autoIncr {
			return "BIGINT AUTO_INCREMENT"
		}
		return "BIGINT"
	case reflect.Uint, reflect.Uint32:
		return "INT UNSIGNED"
	case reflect.Uint8:
		return "TINYINT UNSIGNED"
	case reflect.Uint16:
		return "SMALLINT UNSIGNED"
	case reflect.Uint64:
		return "BIGINT UNSIGNED"
	case reflect.Float32:
		return "FLOAT"
	case reflect.Float64:
		if f.precision > 0 {
			return "DECIMAL(" + strconv.Itoa(f.precision) + "," + strconv.Itoa(f.scale) + ")"
		}
		return "DOUBLE"
	case reflect.String:
		if f.size > 0 {
			if f.size > 16383 {
				return "TEXT"
			}
			return "VARCHAR(" + strconv.Itoa(f.size) + ")"
		}
		return "TEXT"
	}

	// 处理特殊类型
	typeName := f.typ.String()

	// 处理sql.NullXXX类型
	if strings.HasPrefix(typeName, "sql.Null") {
		switch typeName {
		case "sql.NullString":
			if f.size > 0 {
				if f.size > 16383 {
					return "TEXT"
				}
				return "VARCHAR(" + strconv.Itoa(f.size) + ")"
			}
			return "TEXT"
		case "sql.NullInt64":
			return "BIGINT"
		case "sql.NullFloat64":
			return "DOUBLE"
		case "sql.NullBool":
			return "TINYINT(1)"
		case "sql.NullTime":
			return "DATETIME"
		}
	} else if typeName == "time.Time" {
		return "DATETIME"
	}

	// 默认类型
	return "TEXT"
}

func init() {
	RegisterDialect("mysql", &Mysql{})
}