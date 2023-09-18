package utils

import (
	"reflect"
	"strings"
)

type StructToMapInterface interface {
	ToConvert(option ...Option) map[string]interface{}
}

type StructToMap struct {
	ConvertStruct interface{}
}

func NewStructToMap(convertStruct interface{}) *StructToMap {
	return &StructToMap{
		ConvertStruct: convertStruct,
	}
}

// ToConvert 执行struct到map的数据转换
func (o *StructToMap) ToConvert(option ...Option) map[string]interface{} {
	toConvertOption := &ToConvertOption{
		excludeFields: defaultExcludeFields,
		firstLower:    true,
	}
	// 遍历传入的可选参数函数并执行
	for _, opt := range option {
		opt(toConvertOption)
	}

	return o.parseStruct(toConvertOption, nil)
}

func (o *StructToMap) parseStruct(convOption *ToConvertOption, child interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	var val reflect.Value
	if child != nil {
		val = reflect.ValueOf(child)
	} else {
		val = reflect.ValueOf(o.ConvertStruct)
	}
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return result
	}

	var efSet *GenericsSet[string]
	if child == nil {
		efSet = NewGenericsSet(convOption.excludeFields)
	} else {
		efSet = NewSet[string]()
	}

	// 识别类型转换
	relType := val.Type()
	for i := 0; i < relType.NumField(); i++ {
		field := relType.Field(i)
		value := val.Field(i)
		name := field.Name
		if !efSet.Contains(name) {
			tag := field.Tag.Get("json")
			if tag != "" {
				index := strings.Index(tag, ",")
				if index == -1 {
					name = tag
				} else {
					name = tag[:index]
				}
			} else if convOption.firstLower {
				name = FirstLower(name)
			}

			switch value.Kind() {
			case reflect.Array, reflect.Slice:
				tLen := value.Len()
				loopRun := false
				if tLen > 0 {
					firstItem := value.Index(0)
					structName := reflect.TypeOf(firstItem.Interface()).Name()
					if firstItem.Type().Kind() == reflect.Struct && structName != "Decimal" {
						loopRun = true
					}
				}

				if loopRun {
					childResult := make([]map[string]interface{}, 0)
					for j := 0; j < tLen; j++ {
						item := value.Index(j)
						childResult = append(childResult, o.parseStruct(convOption, item.Interface()))
					}
					result[name] = childResult
				} else {
					result[name] = value.Interface()
				}
			case reflect.Struct:
				structName := reflect.TypeOf(value.Interface()).Name()
				if structName == "Decimal" {
					result[name] = value.Interface()
				} else {
					result[name] = o.parseStruct(convOption, value.Interface())
				}
			default:
				result[name] = value.Interface()
			}
		}
	}
	return result
}

var (
	defaultExcludeFields []string
)

type Option func(option *ToConvertOption)

type ToConvertOption struct {
	excludeFields []string
	firstLower    bool
}

// WithExcludeFields 手动追加排除字段，不使用json配置排除，实现多次转换每次都可生成不一样的结构
func WithExcludeFields(excludeFields []string) Option {
	return func(option *ToConvertOption) {
		option.excludeFields = excludeFields
	}
}

// WithFirstLower 首字母是否小写，如果json中配置了格式此配置将失效
func WithFirstLower(firstLower bool) Option {
	return func(option *ToConvertOption) {
		option.firstLower = firstLower
	}
}
