package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
	"reflect"
)

type model struct {
	table string
	// fieldsMap负责原数据名称到数据库列名的映射
	fieldsMap map[string]*field
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
	for i := 0; i < num; i++ {
		f := typ.Field(i)
		fields[f.Name] = &field{colName: utils.CamelToSnake(f.Name)}
	}

	return &model{
		table:     utils.CamelToSnake(typ.Name()),
		fieldsMap: fields,
	}, nil
}
