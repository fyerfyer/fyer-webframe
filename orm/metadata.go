package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
	"reflect"
	"strings"
)

type model struct {
	table string
	// fieldsMap负责原数据名称到数据库列名的映射
	fieldsMap  map[string]*field
	colNameMap map[string]string
}

type field struct {
	colName string
}

func parseModel(v any) (*model, error) {
	typ := reflect.TypeOf(v)

	// 如果是指针类型，获取其元素类型
	// 只支持一重指针
	if typ.Kind() != reflect.Struct {
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
		table:      utils.CamelToSnake(typ.Name()),
		fieldsMap:  fields,
		colNameMap: colNameMap, // 添加这个字段
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
