package simpleexcel

import (
	"reflect"
	"testing"
)

func TestConvertToDynamicData_ShouldValidDynamicObject(t *testing.T) {
	type Product struct {
		ID       int
		Category string
		MetaData map[string]interface{}
	}
	testCases := map[string]struct {
		input  interface{}
		output interface{}
	}{
		"struct with map": {
			input: Product{
				ID:       101,
				Category: "Electronics",
				MetaData: map[string]interface{}{"Brand": "GoLang", "Status": "Available"},
			},
			output: map[string]interface{}{
				"ID":              101,
				"Category":        "Electronics",
				"MetaData_Brand":  "GoLang",
				"MetaData_Status": "Available",
			},
		},
		"slice of structs": {
			input: []Product{
				{
					ID:       101,
					Category: "Electronics",
					MetaData: map[string]interface{}{"Brand": "GoLang", "Status": "Available"},
				},
				{
					ID:       102,
					Category: "Electronics",
					MetaData: map[string]interface{}{"Brand": "GoLang", "Status": "NotAvailable"},
				},
			},
			output: []map[string]interface{}{
				{
					"ID":              101,
					"Category":        "Electronics",
					"MetaData_Brand":  "GoLang",
					"MetaData_Status": "Available",
				},
				{
					"ID":              102,
					"Category":        "Electronics",
					"MetaData_Brand":  "GoLang",
					"MetaData_Status": "NotAvailable",
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			flattenedObject, err := ConvertToDynamicData(tc.input)
			if err != nil {
				t.Fatalf("ConvertToDynamicData failed: %v", err)
			}
			if flattenedObject == nil {
				t.Fatalf("ConvertToDynamicData failed: flattenedObject is nil")
			}
			if !reflect.DeepEqual(flattenedObject, tc.output) {
				t.Errorf("Expected %v, got %v", tc.output, flattenedObject)
			}
		})
	}

}

func TestConvertToDynamicData_DynamicObjectShouldValidValue(t *testing.T) {
	type Product struct {
		ID       int
		Category string
		MetaData map[string]interface{}
	}
	testCases := map[string]struct {
		input        interface{}
		outputFields []map[string]interface{}
	}{
		"struct with map": {
			input: Product{
				ID:       101,
				Category: "Electronics",
				MetaData: map[string]interface{}{"Brand": "GoLang", "Status": "Available"},
			},
			outputFields: []map[string]interface{}{
				{
					"ID":              101,
					"Category":        "Electronics",
					"MetaData_Brand":  "GoLang",
					"MetaData_Status": "Available",
				},
			},
		},
		"slice of structs": {
			input: []Product{
				{
					ID:       101,
					Category: "Electronics",
					MetaData: map[string]interface{}{"Brand": "GoLang", "Status": "Available"},
				},
				{
					ID:       102,
					Category: "Electronics",
					MetaData: map[string]interface{}{"Brand": ".NET", "Status": "NotAvailable"},
				},
			},
			outputFields: []map[string]interface{}{
				{
					"ID":              101,
					"Category":        "Electronics",
					"MetaData_Brand":  "GoLang",
					"MetaData_Status": "Available",
				},
				{
					"ID":              102,
					"Category":        "Electronics",
					"MetaData_Brand":  ".NET",
					"MetaData_Status": "NotAvailable",
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			inputValue := reflect.ValueOf(tc.input)
			if inputValue.Kind() == reflect.Struct {
				// Handle single struct case
				flattenedObject, err := ConvertToDynamicData(tc.input)
				if err != nil {
					t.Fatalf("ConvertToDynamicData failed: %v", err)
				}
				if flattenedObject == nil {
					t.Fatalf("ConvertToDynamicData failed: flattenedObject is nil")
				}
				for _, expected := range tc.outputFields {
					for expectedFieldName, expectedFieldValue := range expected {
						flattenedMap, ok := flattenedObject.(map[string]interface{})
						if ok {
							if !reflect.DeepEqual(flattenedMap[expectedFieldName], expectedFieldValue) {
								t.Errorf("Expected %v, got %v", expectedFieldValue, flattenedMap[expectedFieldName])
							}
						}
					}
				}

			} else if inputValue.Kind() == reflect.Slice {
				// Handle slice of structs case
				flattenedObjects, err := ConvertToDynamicData(tc.input)
				if err != nil {
					t.Fatalf("ConvertToDynamicData failed: %v", err)
				}
				if flattenedObjects == nil {
					t.Fatalf("ConvertToDynamicData failed: flattenedObjects is nil")
				}
				flattenedMapSlice, ok := flattenedObjects.([]map[string]interface{})
				if ok {
					for i, expected := range tc.outputFields {
						for expectedFieldName, expectedFieldValue := range expected {
							if !reflect.DeepEqual(flattenedMapSlice[i][expectedFieldName], expectedFieldValue) {
								t.Errorf("Expected %v, got %v", expectedFieldValue, flattenedMapSlice[i][expectedFieldName])
							}
						}
					}
				} else {
					t.Fatalf("ConvertToDynamicData returned unexpected type: %T", flattenedObjects)
				}

			} else {
				t.Fatalf("Unsupported input type: %T", tc.input)
			}

		})
	}

}
