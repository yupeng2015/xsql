package xsql

import (
	"database/sql"
	"errors"
	"fmt"
	ora "github.com/sijms/go-ora/v2"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Fetcher struct {
	R       *sql.Rows
	Log     *Log
	Options *Options
}

func (t *Fetcher) First(i interface{}) error {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr {
		return errors.New("sql: argument can only be pointer type")
	}
	root := value.Elem()

	rows, err := t.Rows()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return sql.ErrNoRows
	}
	t.ParseStruct(root, rows, 0)
	return nil
}

func (t *Fetcher) Find(i interface{}) error {
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Ptr {
		return errors.New("sql: argument can only be pointer type")
	}
	root := value.Elem()
	itemType := root.Type().Elem()

	rows, err := t.Rows()
	if err != nil {
		return err
	}

	for r := 0; r < len(rows); r++ {
		newItem := reflect.New(itemType)
		if newItem.Kind() == reflect.Ptr {
			newItem = newItem.Elem()
		}
		err = t.ParseStruct(newItem, rows, r)
		if err != nil {
			return err
		}
		root.Set(reflect.Append(root, newItem))
	}

	return nil
}

/*
@Description: 解析结构体映射数据 支持递归
@receiver t
@param newItem
@param rows
@param r
@return error
*/
func (t *Fetcher) ParseStruct(newItem reflect.Value, rows []Row, r int) error {
	for n := 0; n < newItem.NumField(); n++ {
		field := newItem.Field(n)
		fieldType := field.Kind()
		if fieldType == reflect.Struct { //判断如果是结构体
			t.ParseStruct(field, rows, r)
		}
		if !field.CanSet() {
			continue
		}
		tag := newItem.Type().Field(n).Tag.Get("xsql")
		if tag == "-" || tag == "_" {
			continue
		}

		strs := strings.Split(tag, ",")
		//是否空值忽略该字段
		//var omitempy bool
		//if len(strs) > 1{
		//	for _, s := range strs[1:] {
		//		if strings.Contains(s,"omitempty"){
		//			omitempy = true
		//		}
		//	}
		//}
		//if !omitempy{
		//	fields = append(fields, strs[0])
		//}else {
		//	continue
		//}
		tag = strs[0]
		if !rows[r].Exist(tag) {
			continue
		}
		if err := mapped(field, rows[r], tag, t.Options); err != nil {
			return err
		}
	}
	return nil
}

func (t *Fetcher) Rows() ([]Row, error) {
	var debugFunc DebugFunc
	if t.Options.DebugFunc != nil {
		debugFunc = t.Options.DebugFunc
	}

	// 获取列名
	columns, err := t.R.Columns()
	if err != nil {
		return nil, err
	}

	// Make a slice for the values
	values := make([]interface{}, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	var rows []Row

	for t.R.Next() {
		err = t.R.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, value := range values {
			// Here we can check if the value is nil (NULL value)
			if value != nil {
				rowMap[columns[i]] = value
			}
		}

		rows = append(rows, Row{
			v:       rowMap,
			options: t.Options,
		})
	}

	if debugFunc != nil {
		t.Log.RowsAffected = int64(len(rows))
		debugFunc(t.Log)
	}

	return rows, nil
}

type Row struct {
	v       map[string]interface{}
	options *Options
}

func (t Row) Exist(field string) bool {
	_, ok := t.v[field]
	return ok
}

func (t Row) Get(field string) *RowResult {
	if v, ok := t.v[field]; ok {
		return &RowResult{
			v:       v,
			options: t.options,
		}
	}
	return &RowResult{
		v:       "",
		options: t.options,
	}
}

func (t Row) Value() map[string]interface{} {
	return t.v
}

type RowResult struct {
	v       interface{}
	options *Options
}

func (t *RowResult) Empty() bool {
	if b, ok := t.v.([]uint8); ok {
		return len(b) == 0
	}
	if s, ok := t.v.(string); ok {
		return len(s) == 0
	}
	if t.v == nil {
		return true
	}
	return false
}

func (t *RowResult) String() string {
	switch reflect.ValueOf(t.v).Kind() {
	case reflect.Int:
		i := t.v.(int)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Int8:
		i := t.v.(int8)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Int16:
		i := t.v.(int16)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Int32:
		i := t.v.(int32)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Int64:
		i := t.v.(int64)
		return strconv.FormatInt(i, 10)
	case reflect.Uint:
		i := t.v.(uint)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Uint8:
		i := t.v.(uint8)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Uint16:
		i := t.v.(uint16)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Uint32:
		i := t.v.(uint32)
		return strconv.FormatInt(int64(i), 10)
	case reflect.Uint64:
		i := t.v.(uint64)
		return strconv.FormatInt(int64(i), 10)
	case reflect.String:
		return t.v.(string)
	default:
		if b, ok := t.v.([]uint8); ok {
			return string(b)
		}
	}
	return ""
}

func (t *RowResult) Int() int64 {
	switch reflect.ValueOf(t.v).Kind() {
	case reflect.Int:
		i := t.v.(int)
		return int64(i)
	case reflect.Int8:
		i := t.v.(int8)
		return int64(i)
	case reflect.Int16:
		i := t.v.(int16)
		return int64(i)
	case reflect.Int32:
		i := t.v.(int32)
		return int64(i)
	case reflect.Int64:
		i := t.v.(int64)
		return i
	case reflect.Uint:
		i := t.v.(uint)
		return int64(i)
	case reflect.Uint8:
		i := t.v.(uint8)
		return int64(i)
	case reflect.Uint16:
		i := t.v.(uint16)
		return int64(i)
	case reflect.Uint32:
		i := t.v.(uint32)
		return int64(i)
	case reflect.Uint64:
		i := t.v.(uint64)
		return int64(i)
	case reflect.String:
		s := t.v.(string)
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0
		}
		return i
	default:
		if b, ok := t.v.([]uint8); ok {
			s := string(b)
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0
			}
			return i
		}
	}
	return 0
}

func (t *RowResult) Time() time.Time {
	timeLayout := DefaultTimeLayout
	if t.options.TimeLayout != "" {
		timeLayout = t.options.TimeLayout
	}

	typ := t.Type()
	if typ == "string" || typ == "[]uint8" {
		tt, _ := time.ParseInLocation(timeLayout, t.String(), time.Local)
		return tt
	}
	if typ == "time.Time" {
		return t.v.(time.Time)
	}
	if typ == "ora.TimeStamp" {
		return time.Time(t.v.(ora.TimeStamp))
	}
	return time.Time{}
}

func (t *RowResult) Value() interface{} {
	return t.v
}

func (t *RowResult) Type() string {
	return reflect.TypeOf(t.v).String()
}

func mapped(field reflect.Value, row Row, tag string, opts *Options) (err error) {
	timeLayout := DefaultTimeLayout
	if opts.TimeLayout != "" {
		timeLayout = opts.TimeLayout
	}

	res := row.Get(tag)
	v := res.Value()
	fk := field.Kind()
	if fk == reflect.Struct {
		fkStr := field.Type().String()
		switch fkStr {
		case "sql.NullString":
			v = sql.NullString{
				Valid:  true,
				String: res.String(),
			}
			break
		case "sql.NullInt64":
			v = sql.NullInt64{
				Valid: true,
				Int64: res.Int(),
			}
			break
		}
	} else {
		switch field.Kind() {
		case reflect.Int:
			v = int(res.Int())
			break
		case reflect.Int8:
			v = int8(res.Int())
			break
		case reflect.Int16:
			v = int16(res.Int())
			break
		case reflect.Int32:
			v = int32(res.Int())
			break
		case reflect.Int64:
			v = res.Int()
			break
		case reflect.Uint:
			v = uint(res.Int())
			break
		case reflect.Uint8:
			v = uint8(res.Int())
			break
		case reflect.Uint16:
			v = uint16(res.Int())
			break
		case reflect.Uint32:
			v = uint32(res.Int())
			break
		case reflect.Uint64:
			v = uint64(res.Int())
			break
		case reflect.String:
			v = res.String()
			break
		default:
			if !res.Empty() &&
				field.Type().String() == "time.Time" &&
				reflect.ValueOf(v).Type().String() != "time.Time" {
				//fmt.Println(res.String())
				if t, e := time.ParseInLocation(timeLayout, res.String(), time.Local); e == nil {
					v = t
				} else {
					return fmt.Errorf("time parse fail for field %s: %v", tag, e)
				}
			}
		}
	}

	// 追加异常信息
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("type mismatch for field %s: %v", tag, e)
		}
	}()
	field.Set(reflect.ValueOf(v))

	return
}
