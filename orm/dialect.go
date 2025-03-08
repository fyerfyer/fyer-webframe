package orm

import (
	"reflect"
	"strconv"
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

	// DDL相关方法
	CreateTableSQL(m *model) string
	AlterTableSQL(m *model, existingTable *model) string
	TableExistsSQL(schema, table string) string
	ColumnType(f *field) string
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

// 创建表的SQL语句通用实现
func (b *BaseDialect) CreateTableSQL(m *model) string {
	var builder strings.Builder
	builder.WriteString("CREATE TABLE ")
	builder.WriteString(b.Quote(m.table))
	builder.WriteString(" (\n")

	// 添加列定义
	var primaryKeys []string
	var uniques []string
	var indexes []string

	i := 0
	for _, f := range m.fieldsMap {
		if i > 0 {
			builder.WriteString(",\n")
		}

		// 列名和类型
		builder.WriteString("  ")
		builder.WriteString(b.Quote(f.colName))
		builder.WriteString(" ")
		builder.WriteString(b.ColumnType(f))

		// 约束
		if !f.nullable {
			builder.WriteString(" NOT NULL")
		}

		if f.default_ != "" {
			builder.WriteString(" DEFAULT ")
			builder.WriteString(f.default_)
		}

		if f.autoIncr {
			builder.WriteString(" AUTO_INCREMENT")
		}

		if f.comment != "" {
			builder.WriteString(" COMMENT '")
			builder.WriteString(f.comment)
			builder.WriteString("'")
		}

		// 收集约束信息
		if f.primaryKey {
			primaryKeys = append(primaryKeys, f.colName)
		}

		if f.unique {
			uniques = append(uniques, f.colName)
		}

		if f.index {
			indexes = append(indexes, f.colName)
		}

		i++
	}

	// 添加主键约束
	if len(primaryKeys) > 0 {
		builder.WriteString(",\n  PRIMARY KEY (")
		for i, pk := range primaryKeys {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(b.Quote(pk))
		}
		builder.WriteString(")")
	}

	// 添加唯一约束
	for _, colName := range uniques {
		builder.WriteString(",\n  UNIQUE KEY ")
		builder.WriteString(b.Quote("uk_" + m.table + "_" + colName))
		builder.WriteString(" (")
		builder.WriteString(b.Quote(colName))
		builder.WriteString(")")
	}

	// 添加索引
	for _, colName := range indexes {
		builder.WriteString(",\n  KEY ")
		builder.WriteString(b.Quote("idx_" + m.table + "_" + colName))
		builder.WriteString(" (")
		builder.WriteString(b.Quote(colName))
		builder.WriteString(")")
	}

	builder.WriteString("\n)")

	return builder.String()
}

// AlterTableSQL 生成修改表的SQL语句
func (b *BaseDialect) AlterTableSQL(m *model, existingTable *model) string {
	var builder strings.Builder
	builder.WriteString("ALTER TABLE ")
	builder.WriteString(b.Quote(m.table))

	// 处理新增列
	addColumns := []string{}
	alterColumns := []string{}

	for name, newField := range m.fieldsMap {
		if oldField, exists := existingTable.fieldsMap[name]; !exists {
			// 新增列
			addSql := "\n  ADD COLUMN " + b.Quote(newField.colName) + " " + b.ColumnType(newField)
			if !newField.nullable {
				addSql += " NOT NULL"
			}
			if newField.default_ != "" {
				addSql += " DEFAULT " + newField.default_
			}
			addColumns = append(addColumns, addSql)
		} else if b.ColumnType(newField) != b.ColumnType(oldField) ||
			newField.nullable != oldField.nullable ||
			newField.default_ != oldField.default_ {
			// 修改列
			alterSql := "\n  MODIFY COLUMN " + b.Quote(newField.colName) + " " + b.ColumnType(newField)
			if !newField.nullable {
				alterSql += " NOT NULL"
			}
			if newField.default_ != "" {
				alterSql += " DEFAULT " + newField.default_
			}
			alterColumns = append(alterColumns, alterSql)
		}
	}

	// 组合所有变更
	changes := append(addColumns, alterColumns...)
	for i, change := range changes {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(change)
	}

	return builder.String()
}

// TableExistsSQL 生成检查表是否存在的SQL
func (b *BaseDialect) TableExistsSQL(schema, table string) string {
	return "SELECT 1 FROM information_schema.tables WHERE table_name = '" + table + "'"
}

// ColumnType 根据Go类型确定SQL类型
func (b *BaseDialect) ColumnType(f *field) string {
	// 如果字段明确指定了SQL类型，直接使用
	if f.sqlType != "" {
		return f.sqlType
	}

	switch f.typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int32:
		return "INTEGER"
	case reflect.Int8:
		return "TINYINT"
	case reflect.Int16:
		return "SMALLINT"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint32:
		return "INTEGER UNSIGNED"
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
			return "VARCHAR(" + strconv.Itoa(f.size) + ")"
		}
		return "TEXT"
	}

	// 处理复合类型或特殊类型
	typeName := f.typ.String()

	// sql.NullString等特殊处理
	if strings.HasPrefix(typeName, "sql.Null") {
		switch typeName {
		case "sql.NullString":
			if f.size > 0 {
				return "VARCHAR(" + strconv.Itoa(f.size) + ")"
			}
			return "TEXT"
		case "sql.NullInt64":
			return "BIGINT"
		case "sql.NullFloat64":
			return "DOUBLE"
		case "sql.NullBool":
			return "BOOLEAN"
		case "sql.NullTime":
			return "DATETIME"
		}
	} else if typeName == "time.Time" {
		return "DATETIME"
	}

	// 默认
	return "TEXT"
}