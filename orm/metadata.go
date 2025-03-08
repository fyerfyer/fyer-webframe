package orm

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
)

type model struct {
	table         string
	fieldsMap     map[string]*field
	colNameMap    map[string]string
	colAliasMap   map[string]bool
	tableAliasMap map[string]string
	dialect       Dialect // 添加dialect字段
	index         int     // 用于postgresql的占位符
}

// field 扩展字段结构体，添加更多类型和约束信息
type field struct {
	colName    string
	typ        reflect.Type  // 字段类型
	size       int           // 字段大小，如varchar(255)中的255
	nullable   bool          // 是否允许为空
	primaryKey bool          // 是否为主键
	unique     bool          // 是否唯一
	index      bool          // 是否索引
	default_   string        // 默认值
	comment    string        // 字段注释
	precision  int           // 精度(小数点后位数)
	scale      int           // 范围(总位数)
	autoIncr   bool          // 是否自增
	sqlType    string        // 显式指定的SQL类型
}

func parseModel(v any) (*model, error) {
	typ := reflect.TypeOf(v)

	// 如果是指针类型，获取其元素类型
	// 只支持一重指针
	for typ.Kind() != reflect.Struct {
		typ = typ.Elem()
	}

	num := typ.NumField()
	fields := make(map[string]*field, num)
	colNameMap := make(map[string]string, num)

	for i := 0; i < num; i++ {
		fieldVar := &field{}
		f := typ.Field(i)

		// 记录字段类型信息
		fieldVar.typ = f.Type

		// 检查是否有自定义tag
		tags, err := parseTag(f)
		if err != nil {
			return nil, err
		}

		// 设置列名
		if colName, ok := tags["column_name"]; ok {
			fieldVar.colName = colName
		} else {
			fieldVar.colName = utils.CamelToSnake(f.Name)
		}

		// 解析其他标签属性
		fieldVar.primaryKey = tags["primary_key"] == "true"
		fieldVar.nullable = tags["nullable"] != "false" // 默认可空
		fieldVar.unique = tags["unique"] == "true"
		fieldVar.index = tags["index"] == "true"
		fieldVar.autoIncr = tags["auto_increment"] == "true" || tags["auto_incr"] == "true"
		fieldVar.default_ = tags["default"]
		fieldVar.comment = tags["comment"]

		if size, ok := tags["size"]; ok {
			fieldVar.size, _ = strconv.Atoi(size)
		}

		if precision, ok := tags["precision"]; ok {
			fieldVar.precision, _ = strconv.Atoi(precision)
		}

		if scale, ok := tags["scale"]; ok {
			fieldVar.scale, _ = strconv.Atoi(scale)
		}

		if sqlType, ok := tags["type"]; ok {
			fieldVar.sqlType = sqlType
		}

		fields[f.Name] = fieldVar
		// 存储列名到字段名的映射
		colNameMap[fieldVar.colName] = f.Name
	}

	return &model{
		table:         utils.CamelToSnake(typ.Name()),
		fieldsMap:     fields,
		colNameMap:    colNameMap,
		colAliasMap:   make(map[string]bool, 4),
		tableAliasMap: make(map[string]string, 4),
		dialect:       nil, // 初始为nil，将在后续设置
	}, nil
}

// parseTag 解析tag
// tag格式：`orm:"column_name:col_name;primary_key:true;size:255"`
func parseTag(field reflect.StructField) (map[string]string, error) {
	tag := field.Tag.Get("orm")
	if tag == "" {
		return nil, nil
	}

	tags := make(map[string]string, 8)
	for _, part := range strings.Split(tag, ";") {
		if part == "" {
			continue
		}

		kvs := strings.Split(part, ":")
		if len(kvs) == 1 {
			// 处理没有值的标签，如 `orm:"primary_key"`
			tags[kvs[0]] = "true"
		} else if len(kvs) == 2 {
			tags[kvs[0]] = kvs[1]
		} else {
			return nil, ferr.ErrInvalidTag(tag)
		}
	}

	return tags, nil
}

// SetDialect 为模型设置方言
func (m *model) SetDialect(dialect Dialect) {
	m.dialect = dialect
}

// GetTableName 获取表名
func (m *model) GetTableName() string {
	return m.table
}

// GetPrimaryKey 获取主键字段
func (m *model) GetPrimaryKey() (string, bool) {
	for name, field := range m.fieldsMap {
		if field.primaryKey {
			return name, true
		}
	}
	return "", false
}