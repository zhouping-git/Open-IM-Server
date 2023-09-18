package utils

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"reflect"
	"strings"
)

func MapToStruct(data map[string]interface{}, target interface{}) {
	_value := reflect.ValueOf(target).Elem()
	_type := reflect.TypeOf(target).Elem()

	nameMap := make(map[string]int)
	for i := 0; i < _value.NumField(); i++ {
		key := ""
		tag := _type.Field(i).Tag.Get("json")
		if tag != "" {
			index := strings.Index(tag, ",")
			if index == -1 {
				key = tag
			} else {
				key = tag[:index]
			}
		} else {
			key = FirstLower(_type.Field(i).Name)
		}
		nameMap[key] = i
	}

	for k, v := range data {
		if index, ok := nameMap[k]; ok {
			// 检测变量类型，要判断struct属性的类型，是否是指针
			if _value.Field(index).Kind() == reflect.Ptr &&
				_type.Field(index).Type.String() == "*"+reflect.TypeOf(v).String() {
				switch _value.Field(index).Kind() {
				case reflect.Int:
					tempVal := v.(int)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Int8:
					tempVal := v.(int8)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Int16:
					tempVal := v.(int16)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Int32:
					tempVal := v.(int32)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Int64:
					tempVal := v.(int64)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Float32:
					tempVal := v.(float32)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Float64:
					tempVal := v.(float64)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Bool:
					tempVal := v.(bool)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.String:
					tempVal := v.(string)
					_value.Field(index).Set(reflect.ValueOf(&tempVal))
				case reflect.Array, reflect.Slice:
					tempVal := v.(string)
					if tempVal != "null" {

					}
				case reflect.Struct:
					structName := reflect.TypeOf(_value.Field(index).Interface()).Name()
					if structName == "Decimal" {
						tempVal, err := decimal.NewFromString(v.(string))
						if err == nil {
							_value.Field(index).Set(reflect.ValueOf(&tempVal))
						}
					}
				}
			} else if _type.Field(index).Type.String() == reflect.TypeOf(v).String() {
				_value.Field(index).Set(reflect.ValueOf(v))
			} else {
				panic(errors.New(fmt.Sprintf("key「%v」value %v」; type 「%v」 => 「%v」\n",
					k,
					v,
					reflect.TypeOf(v).String(),
					_type.Field(index).Type.String())))
			}
		}
	}

	//for k, v := range data {
	//	if index, ok := nameMap[k]; ok {
	//		// 检测变量类型，要判断struct属性的类型，是否是指针
	//		if _value.Field(index).Kind() == reflect.Ptr &&
	//			_type.Field(index).Type.String() == "*"+reflect.TypeOf(v).String() {
	//			switch v.(type) {
	//			case int:
	//				tempVal := v.(int)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case int8:
	//				tempVal := v.(int8)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case int16:
	//				tempVal := v.(int16)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case int32:
	//				tempVal := v.(int32)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case int64:
	//				tempVal := v.(int64)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case float32:
	//				tempVal := v.(float32)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case float64:
	//				tempVal := v.(float64)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case bool:
	//				tempVal := v.(bool)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			case string:
	//				tempVal := v.(string)
	//				_value.Field(index).Set(reflect.ValueOf(&tempVal))
	//			}
	//		} else if _type.Field(index).Type.String() == reflect.TypeOf(v).String() {
	//			_value.Field(index).Set(reflect.ValueOf(v))
	//		} else {
	//			panic(errors.New(fmt.Sprintf("key「%v」value %v」; type 「%v」 => 「%v」\n",
	//				k,
	//				v,
	//				reflect.TypeOf(v).String(),
	//				_type.Field(index).Type.String())))
	//		}
	//	}
	//}
}
