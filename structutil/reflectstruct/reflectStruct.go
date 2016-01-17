package reflectstruct

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/tsaikd/KDGoLib/errutil"
	"github.com/tsaikd/KDGoLib/reflectutil"
)

// expose errors
var (
	ErrorUnknownSourceMapKeyType1       = errutil.ErrorFactory("unknown source map key type: %v")
	ErrorUnsupportedReflectFieldMethod2 = errutil.ErrorFactory("unsupported reflect field method: %v <- %v")
)

func unsafeReflectFieldSlice2Slice(field reflect.Value, val reflect.Value) (err error) {
	if field.Type().Elem().Kind() == val.Type().Elem().Kind() {
		field.Set(val)
	} else {
		len := val.Len()
		vals := reflect.MakeSlice(field.Type(), len, len)
		for i := 0; i < len; i++ {
			if err = reflectField(vals.Index(i), val.Index(i)); err != nil {
				return
			}
		}
		field.Set(vals)
	}
	return
}

var (
	jsonUnmarshaler = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	typeOfBytes     = reflect.TypeOf([]byte(nil))
)

func reflectField(field reflect.Value, val reflect.Value) (err error) {
	if field.Type().Implements(jsonUnmarshaler) {
		switch val.Kind() {
		case reflect.String:
			return json.Unmarshal([]byte(`"`+val.String()+`"`), field.Interface())
		case reflect.Slice:
			if val.Type() == typeOfBytes {
				return json.Unmarshal(val.Bytes(), field.Interface())
			}
		}
	}

	if field.CanAddr() && field.Addr().Type().Implements(jsonUnmarshaler) {
		switch val.Kind() {
		case reflect.String:
			return reflectField(field.Addr(), val)
		case reflect.Slice:
			if val.Type() == typeOfBytes {
				return reflectField(field.Addr(), val)
			}
		}
	}

	// get val real type
	valElemType := val.Type()
	switch valElemType.Kind() {
	case reflect.Ptr:
		valElemType = valElemType.Elem()
		val = val.Elem()
	case reflect.Interface:
		val = val.Elem()
	}

	// field is not slice, val is slice, assign last val
	if field.Kind() != reflect.Slice && val.Kind() == reflect.Slice {
		lastidx := val.Len() - 1
		if lastidx >= 0 {
			return reflectField(field, val.Index(lastidx))
		}
		return
	}

	switch field.Kind() {
	case reflect.Ptr:
		return reflectField(field.Elem(), val)
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		switch val.Kind() {
		case reflect.String:
			valstr := val.String()
			v, err := strconv.ParseInt(valstr, 0, 64)
			if err != nil {
				return err
			}
			field.SetInt(v)
			return err
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			field.SetInt(val.Int())
			return
		case reflect.Float64, reflect.Float32:
			field.SetInt(int64(val.Float()))
			return
		case reflect.Bool:
			if val.Bool() {
				field.SetInt(1)
			} else {
				field.SetInt(0)
			}
			return
		}
	case reflect.Float64, reflect.Float32:
		switch val.Kind() {
		case reflect.String:
			valstr := val.String()
			v, err := strconv.ParseFloat(valstr, 64)
			if err != nil {
				return err
			}
			field.SetFloat(v)
			return err
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			field.SetFloat(float64(val.Int()))
			return
		case reflect.Float64, reflect.Float32:
			field.SetFloat(val.Float())
			return
		case reflect.Bool:
			if val.Bool() {
				field.SetFloat(1)
			} else {
				field.SetFloat(0)
			}
			return
		}
	case reflect.Bool:
		switch val.Kind() {
		case reflect.String:
			valstr := val.String()
			v, err := govalidator.ToBoolean(valstr)
			if err != nil {
				return err
			}
			field.SetBool(v)
			return nil
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			field.SetBool(val.Int() > 0)
			return
		case reflect.Float64, reflect.Float32:
			field.SetBool(val.Float() > 0)
			return
		case reflect.Bool:
			field.SetBool(val.Bool())
			return
		}
	case reflect.String:
		switch val.Kind() {
		case reflect.String:
			field.SetString(val.String())
			return
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			field.SetString(strconv.FormatInt(val.Int(), 10))
			return
		case reflect.Float64, reflect.Float32:
			field.SetString(strconv.FormatFloat(val.Float(), 'f', -1, 64))
			return
		case reflect.Bool:
			field.SetString(strconv.FormatBool(val.Bool()))
			return
		}
	case reflect.Slice:
		switch val.Kind() {
		case reflect.Slice:
			len := val.Len()
			vals := reflect.MakeSlice(field.Type(), len, len)
			for i := 0; i < len; i++ {
				if err = reflectField(vals.Index(i), val.Index(i)); err != nil {
					return
				}
			}
			field.Set(vals)
			return
		}
	case reflect.Struct:
		fieldInfoMap := buildReflectFieldInfo(nil, field)

		switch val.Kind() {
		case reflect.Map:
			for _, valmapkey := range val.MapKeys() {
				switch valmapkey.Kind() {
				case reflect.String:
					mapkey := valmapkey.String()
					if valDestField, ok := fieldInfoMap[mapkey]; ok {
						valmapval := reflectutil.EnsureValue(val.MapIndex(valmapkey))
						if err = reflectField(valDestField, valmapval); err != nil {
							return
						}
					}
				default:
					return ErrorUnknownSourceMapKeyType1.New(nil, valmapkey.Kind())
				}
			}
			return
		case reflect.Struct:
			if val.Type().AssignableTo(field.Type()) {
				field.Set(val)
				return
			}

			valInfoMap := buildReflectFieldInfo(nil, val)
			for key, valField := range valInfoMap {
				fieldInfo, exist := fieldInfoMap[key]
				if !exist {
					continue
				}
				if err = reflectField(fieldInfo, valField); err != nil {
					return
				}
			}
			return
		}
	case reflect.Interface:
		field.Set(val)
		return
	}

	return ErrorUnsupportedReflectFieldMethod2.New(nil, field.Kind(), val.Kind())
}

func buildReflectFieldInfo(fieldInfoMap map[string]reflect.Value, value reflect.Value) map[string]reflect.Value {
	if fieldInfoMap == nil {
		fieldInfoMap = map[string]reflect.Value{}
	}
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		if tag := field.Tag.Get("json"); tag != "" {
			if tagvals := strings.Split(tag, ","); len(tagvals) > 0 && tagvals[0] != "-" {
				fieldInfoMap[tagvals[0]] = value.Field(i)
				continue
			}
		}
		if tag := field.Tag.Get("reflect"); tag == "inherit" {
			childValue := value.Field(i)
			if childValue.Kind() == reflect.Ptr && childValue.IsNil() {
				childValue = reflect.New(childValue.Type().Elem())
				value.Field(i).Set(childValue)
				childValue = childValue.Elem()
			}
			fieldInfoMap = buildReflectFieldInfo(fieldInfoMap, childValue)
			continue
		}
		if field.Name[:1] == strings.ToUpper(field.Name[:1]) {
			fieldInfoMap[field.Name] = value.Field(i)
			continue
		}
	}
	return fieldInfoMap
}

// ReflectStruct reflect data from src to dest
func ReflectStruct(dest interface{}, src interface{}) (err error) {
	if dest == nil || src == nil {
		return
	}

	return reflectField(
		reflect.ValueOf(dest),
		reflect.ValueOf(src),
	)
}
