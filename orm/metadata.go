package orm

import (
	"reflect"
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

type field struct {
	colName string
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

		// 检查是否有自定义tag
		tags, err := parseTag(f)
		if err != nil {
			return nil, err
		}

		if colName, ok := tags["column_name"]; ok {
			fieldVar.colName = colName
		} else {
			fieldVar.colName = utils.CamelToSnake(f.Name)
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
// tag格式：`orm:"column_name:col_name"`
func parseTag(field reflect.StructField) (map[string]string, error) {
	tag := field.Tag.Get("orm")
	if tag == "" {
		return nil, nil
	}

	tags := make(map[string]string, 4)
	for _, tag := range strings.Split(tag, ";") {
		kvs := strings.Split(tag, ",")
		for _, kv := range kvs {
			kvPair := strings.Split(kv, ":")
			if len(kvPair) != 2 {
				return nil, ferr.ErrInvalidTag(tag)
			}
			if kvPair[0] == "column_name" {
				tags["column_name"] = kvPair[1]
			} else {
				return nil, ferr.ErrInvalidTag(tag)
			}
		}
	}

	return tags, nil
}

// SetDialect 为模型设置方言
func (m *model) SetDialect(dialect Dialect) {
	m.dialect = dialect
}
