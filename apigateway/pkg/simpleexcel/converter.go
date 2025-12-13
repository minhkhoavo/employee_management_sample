package simpleexcel

import (
	"fmt"
	"reflect"
)

func ConvertToDynamicData(data interface{}) (interface{}, error) {
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return flattenStruct(val)
	case reflect.Slice:
		return flattenSlice(val)
	default:
		return nil, fmt.Errorf("expected struct or slice, got %v", val.Kind())
	}
}

func flattenStruct(val reflect.Value) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		if field.Kind() == reflect.Map {
			// Flatten map fields with prefix
			if field.IsNil() {
				continue
			}
			for _, key := range field.MapKeys() {
				mapValue := field.MapIndex(key)
				flattenedKey := fmt.Sprintf("%s_%v", fieldName, key.Interface())
				result[flattenedKey] = mapValue.Interface()
			}
		} else {
			// Direct field assignment
			result[fieldName] = field.Interface()
		}
	}

	return result, nil
}

func flattenSlice(val reflect.Value) ([]map[string]interface{}, error) {
	length := val.Len()
	result := make([]map[string]interface{}, length)

	for i := 0; i < length; i++ {
		elem := val.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		if elem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected slice of structs, got slice of %v", elem.Kind())
		}

		flattened, err := flattenStruct(elem)
		if err != nil {
			return nil, err
		}
		result[i] = flattened
	}

	return result, nil
}
