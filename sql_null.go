package xsql

import (
	"database/sql"
	"reflect"
)

type SqlNullKind struct {
	reflect.Kind
}

type SqlNull struct {
	reflect.Value
}

func (this SqlNull) FieldTypeBasic(i int) string {
	nullStr := this.Field(i).Type().String()
	switch nullStr {
	case "sql.NullString":
		return "string"
	case "sql.NullInt64":
		return "int64"
	}
	return this.Field(i).Type().String()
}

func (this SqlNull) FieldAnyBasic(i int) any {
	nullStr := this.Field(i).Type().String()
	var v any
	switch nullStr {
	case "sql.NullString":
		v, _ = this.Field(i).Interface().(sql.NullString).Value()
		if v == nil {
			v = ""
		}
		break
	case "sql.NullInt64":
		v, _ = this.Field(i).Interface().(sql.NullInt64).Value()
		if v == nil {
			v = 0
		}
		break
	default:
		v = this.Field(i).Interface()
	}
	return v
}

//func (this SqlNull) Kind() SqlNullKind {
//
//}

func GetBasicValFromStruct(v any) map[string]any {
	hofvalue := reflect.ValueOf(v)
	hofType := reflect.TypeOf(v)
	m := make(map[string]any)
	for i := 0; i < hofType.NumField(); i++ { //循环结构体内字段的数量
		//获取结构体内索引为i的字段值
		sf := hofType.Field(i)
		//fieldName := sf.Name
		nullStr := sf.Type.String()         //获取类型
		im := hofvalue.Field(i).Interface() //获取该属性实际值
		tag := sf.Tag.Get("json")
		switch nullStr {
		case "sql.NullString":
			m[tag] = im.(sql.NullString).String
			break
		case "sql.NullInt64":
			m[tag] = im.(sql.NullInt64).Int64
			break
		}
	}
	return m
}

/*
@Description: 结构体转map
@param v
@return map[string]any
*/
func Struct2Map(v any) map[string]any {
	hofType := reflect.TypeOf(v)
	hofvalue := reflect.ValueOf(v)
	m := make(map[string]any)
	for i := 0; i < hofType.NumField(); i++ {
		sf := hofType.Field(i)
		tagConvert := sf.Tag.Get("convert")
		tagJson := sf.Tag.Get("json")
		im := hofvalue.Field(i).Interface()
		if tagConvert == "bool" {
			switch im.(type) {
			case int64:
				m[tagJson] = Int2Bool(im.(int64))
				break
			case int:
				m[tagJson] = Int2Bool(im.(int))
				break
			}
		} else if tagConvert == "int" {
			m[tagJson] = Bool2Int(im.(bool))
		}
	}
	return m
}

func Bool2Int(b bool) int {
	if b {
		return 1
	}
	return 0
}

func Int2Bool[T int | int64](n T) bool {
	if n == 0 {
		return false
	}
	return true
}
