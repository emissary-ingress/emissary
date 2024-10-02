package v4alpha1

import (
	"fmt"
	"reflect"
	"strings"
)

// NativeToUnstructured converts a native v4 struct to a
// map[string]interface{} representation that can be easily serialized to JSON
// or YAML, using the given convention ("json" or "v3") to determine the field
// names in the resulting map.
func NativeToUnstructured(v interface{}, convention string) (map[string]interface{}, error) {
	if convention != "json" && convention != "v3" {
		return nil, fmt.Errorf("Invalid convention: %s", convention)
	}

	result := make(map[string]interface{})
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// Loop through the struct fields
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)

		// Get the appropriate tag based on the convention
		fieldName := field.Tag.Get(convention)

		// If the field is a struct, recursively serialize it
		if fieldName != "" {
			if value.Kind() == reflect.Struct {
				nestedResult, err := NativeToUnstructured(value.Interface(), convention)
				if err != nil {
					return nil, err
				}
				result[fieldName] = nestedResult
			} else {
				result[fieldName] = value.Interface()
			}
		}
	}

	return result, nil
}

// UnstructuredToNative converts an unstructured map[string]interface{} to a
// native v4 struct, using the given convention ("json" or "v3") to determine
// the field names in the input map. The "v" input must be a native-type
// variable.
func UnstructuredToNative(data map[string]interface{}, v interface{}, convention string) error {
	if convention != "json" && convention != "v3" {
		return fmt.Errorf("Invalid convention: %s", convention)
	}

	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	fmt.Printf("kind %s, type %s, value %#v\n", typ.Kind(), typ.Name(), val)
	fmt.Printf("data %#v\n", data)

	// Ensure the input is a pointer to a struct
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Invalid input type: %s", typ)
	}

	// Loop through the struct fields
	structType := typ.Elem()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Get the appropriate tag based on the convention
		fieldName := field.Tag.Get(convention)

		if (fieldName == "") && (convention == "v3") {
			// If the field name is empty, try the "json" convention
			fieldName = field.Tag.Get("json")
		}

		// Split the field name on comma and use only the first part
		idx := strings.Index(fieldName, ",")

		if idx != -1 {
			fieldName = fieldName[:idx]
		}

		fmt.Printf("%d: %s - kind %s, type %s, tag %s\n",
			i, field.Name, field.Type.Kind(), field.Type.Name(), fieldName)

		if fieldName != "" {
			if (field.Type.Kind() == reflect.Struct) ||
				(field.Type.Kind() == reflect.Ptr &&
					field.Type.Elem().Kind() == reflect.Struct) {
				// If it's a pointer to a struct, recursively deserialize
				nestedData, ok := data[fieldName].(map[string]interface{})

				if ok {
					nestedStructPtr := reflect.New(field.Type).Interface()

					if err := UnstructuredToNative(nestedData, nestedStructPtr, convention); err != nil {
						return err
					}
					val.Field(i).Set(reflect.ValueOf(nestedStructPtr))
				} else {
					fmt.Printf("  ptr struct is not a map: %#v\n", data[fieldName])
					// return fmt.Errorf("Expected nested struct data for field %s", fieldName)
				}
			} else {
				// Otherwise, set the field value directly
				value, ok := data[fieldName]

				if ok {
					valueOf := reflect.ValueOf(value)

					fmt.Printf("%d:   direct set (canset %#v) to kind %s, type %s, tag %s\n",
						i, field.Type.AssignableTo(valueOf.Type()), valueOf.Kind(), valueOf.Type().Name(), valueOf)

					val.Field(i).Set(valueOf)
				}
			}
		}
	}

	return nil
}
