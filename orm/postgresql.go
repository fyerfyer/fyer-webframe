package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"reflect"
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

// CreateTableSQL 为PostgreSQL生成建表语句
func (p Postgresql) CreateTableSQL(m *model) string {
	// 先调用基本实现生成通用的SQL
	baseSQL := p.BaseDialect.CreateTableSQL(m)
	newSQL := strings.ReplaceAll(baseSQL, "`", "\"")
	// PostgreSQL没有特殊的表选项，所以直接返回基础SQL
	return newSQL + ";"
}

// AlterTableSQL 实现PostgreSQL特定的表结构修改语句
func (p Postgresql) AlterTableSQL(m *model, existingTable *model) string {
	var builder strings.Builder
	builder.WriteString("ALTER TABLE ")
	builder.WriteString(p.Quote(m.table))

	// 处理新增列
	addColumns := []string{}
	alterColumns := []string{}

	for name, newField := range m.fieldsMap {
		if oldField, exists := existingTable.fieldsMap[name]; !exists {
			// 新增列
			addSql := "\n  ADD COLUMN " + p.Quote(newField.colName) + " " + p.ColumnType(newField)
			if !newField.nullable {
				addSql += " NOT NULL"
			}
			if newField.default_ != "" {
				addSql += " DEFAULT " + newField.default_
			}
			addColumns = append(addColumns, addSql)
		} else if p.ColumnType(newField) != p.ColumnType(oldField) ||
			newField.nullable != oldField.nullable {
			// 修改列 - PostgreSQL需要多个命令
			if p.ColumnType(newField) != p.ColumnType(oldField) {
				alterSql := "\n  ALTER COLUMN " + p.Quote(newField.colName) + " TYPE " + p.ColumnType(newField)
				alterColumns = append(alterColumns, alterSql)
			}

			if newField.nullable != oldField.nullable {
				var nullableSql string
				if newField.nullable {
					nullableSql = "\n  ALTER COLUMN " + p.Quote(newField.colName) + " DROP NOT NULL"
				} else {
					nullableSql = "\n  ALTER COLUMN " + p.Quote(newField.colName) + " SET NOT NULL"
				}
				alterColumns = append(alterColumns, nullableSql)
			}

			if newField.default_ != oldField.default_ {
				var defaultSql string
				if newField.default_ == "" {
					defaultSql = "\n  ALTER COLUMN " + p.Quote(newField.colName) + " DROP DEFAULT"
				} else {
					defaultSql = "\n  ALTER COLUMN " + p.Quote(newField.colName) + " SET DEFAULT " + newField.default_
				}
				alterColumns = append(alterColumns, defaultSql)
			}
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

	return builder.String() + ";"
}

// TableExistsSQL 实现PostgreSQL检查表是否存在的SQL
func (p Postgresql) TableExistsSQL(schema, table string) string {
	if schema == "" {
		schema = "public"
	}
	return "SELECT 1 FROM information_schema.tables WHERE table_schema = '" + schema + "' AND table_name = '" + table + "'"
}

// ColumnType 为PostgreSQL实现Go类型到SQL类型的映射
func (p Postgresql) ColumnType(f *field) string {
	// 如果字段明确指定了SQL类型，直接使用
	if f.sqlType != "" {
		return f.sqlType
	}

	// 根据Go类型映射PostgreSQL类型
	switch f.typ.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int32:
		if f.autoIncr {
			return "SERIAL"
		}
		return "INTEGER"
	case reflect.Int8:
		return "SMALLINT"
	case reflect.Int16:
		return "SMALLINT"
	case reflect.Int64:
		if f.autoIncr {
			return "BIGSERIAL"
		}
		return "BIGINT"
	case reflect.Uint, reflect.Uint32:
		return "INTEGER"
	case reflect.Uint8:
		return "SMALLINT"
	case reflect.Uint16:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		if f.precision > 0 {
			return "NUMERIC(" + strconv.Itoa(f.precision) + "," + strconv.Itoa(f.scale) + ")"
		}
		return "DOUBLE PRECISION"
	case reflect.String:
		if f.size > 0 {
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
				return "VARCHAR(" + strconv.Itoa(f.size) + ")"
			}
			return "TEXT"
		case "sql.NullInt64":
			return "BIGINT"
		case "sql.NullFloat64":
			return "DOUBLE PRECISION"
		case "sql.NullBool":
			return "BOOLEAN"
		case "sql.NullTime":
			return "TIMESTAMP WITH TIME ZONE"
		}
	} else if typeName == "time.Time" {
		return "TIMESTAMP WITH TIME ZONE"
	}

	// 默认类型
	return "TEXT"
}

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