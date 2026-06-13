package ytmusicapi

// ReadValue navigates through nested JSON structure using a path of string or int keys
func ReadValue(object interface{}, arrayOfkeys []interface{}) interface{} {
	if len(arrayOfkeys) == 0 {
		return object
	}

	key := arrayOfkeys[0]

	switch key := key.(type) {
	case string:
		if obj, ok := object.(map[string]interface{}); ok {
			return ReadValue(obj[key], arrayOfkeys[1:])
		}
		return nil
	case int:
		if arr, ok := object.([]interface{}); ok && len(arr) > key {
			return ReadValue(arr[key], arrayOfkeys[1:])
		}
		return nil
	default:
		return nil
	}
}

// ReadValueString reads a string value from a nested JSON structure
func ReadValueString(object interface{}, arrayOfkeys []interface{}) string {
	value, _ := ReadValue(object, arrayOfkeys).(string)
	return value
}
