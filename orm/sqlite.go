package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"reflect"
	"strconv"
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

// JulianDay 返回距离公元前4713年1月1日12时的天数
func (s Sqlite) JulianDay(dateExpr string) string {
	return "julianday(" + dateExpr + ")"
}

// CreateTableSQL 为SQLite生成建表语句
func (s Sqlite) CreateTableSQL(m *model) string {
	// 先调用基本实现生成通用的SQL
	baseSQL := s.BaseDialect.CreateTableSQL(m)
	newSQL := strings.ReplaceAll(baseSQL, "`", "\"")
	// SQLite没有特殊的表选项，所以直接返回基础SQL
	return newSQL + ";";
}

// AlterTableSQL 实现SQLite特定的表结构修改语句
func (s Sqlite) AlterTableSQL(m *model, existingTable *model) string {
	// SQLite不支持直接修改列定义，而是需要通过以下步骤：
	// 1. 创建新表 (使用CREATE TABLE)
	// 2. 复制数据 (使用INSERT)
	// 3. 删除旧表 (使用DROP TABLE)
	// 4. 重命名新表 (使用ALTER TABLE RENAME TO)

	var builder strings.Builder

	// 这里实现一个简化版本，只处理添加列
	builder.WriteString("ALTER TABLE ")
	builder.WriteString(s.Quote(m.table))

	// 处理新增列
	for name, newField := range m.fieldsMap {
		if _, exists := existingTable.fieldsMap[name]; !exists {
			// SQLite ALTER TABLE 只支持添加列
			builder.WriteString(" ADD COLUMN ")
			builder.WriteString(s.Quote(newField.colName))
			builder.WriteString(" ")
			builder.WriteString(s.ColumnType(newField))

			if !newField.nullable {
				builder.WriteString(" NOT NULL")
			}

			if newField.default_ != "" {
				builder.WriteString(" DEFAULT ")
				builder.WriteString(newField.default_)
			}

			// 注意：SQLite的ADD COLUMN 不支持添加主键约束
			break // SQLite每次只能添加一列，这里简化处理只添加第一个新列
		}
	}

	return builder.String() + ";";
}

// TableExistsSQL 实现SQLite检查表是否存在的SQL
func (s Sqlite) TableExistsSQL(schema, table string) string {
	return "SELECT 1 FROM sqlite_master WHERE type='table' AND name='" + table + "'";
}

// ColumnType 为SQLite实现Go类型到SQL类型的映射
func (s Sqlite) ColumnType(f *field) string {
	// 如果字段明确指定了SQL类型，直接使用
	if f.sqlType != "" {
		return f.sqlType
	}

	// SQLite只有 NULL, INTEGER, REAL, TEXT, BLOB 5种类型
	// 但为了兼容其他数据库，我们会使用更丰富的类型名
	switch f.typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f.autoIncr {
			return "INTEGER PRIMARY KEY AUTOINCREMENT"
		}
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.String:
		if f.size > 0 {
			return "TEXT(" + strconv.Itoa(f.size) + ")" // 注意SQLite实际上忽略这个大小
		}
		return "TEXT"
	}

	// 处理特殊类型
	typeName := f.typ.String()

	// 处理sql.NullXXX类型
	if strings.HasPrefix(typeName, "sql.Null") {
		switch typeName {
		case "sql.NullString":
			return "TEXT"
		case "sql.NullInt64":
			return "INTEGER"
		case "sql.NullFloat64":
			return "REAL"
		case "sql.NullBool":
			return "BOOLEAN"
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
	RegisterDialect("sqlite", &Sqlite{})
}