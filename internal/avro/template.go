package avro

import (
	"encoding/json"
	"fmt"
)

// GenerateTemplate creates a JSON template from an Avro schema.
// The template contains placeholder values for each field.
func GenerateTemplate(schemaJSON string) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return "", fmt.Errorf("parsing schema: %w", err)
	}

	result, err := generateValue(schema)
	if err != nil {
		return "", err
	}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting template: %w", err)
	}

	return string(pretty), nil
}

func generateValue(schema interface{}) (interface{}, error) {
	switch s := schema.(type) {
	case string:
		return generatePrimitive(s)
	case []interface{}:
		return generateUnion(s)
	case map[string]interface{}:
		return generateComplex(s)
	default:
		return nil, fmt.Errorf("unexpected schema type: %T", schema)
	}
}

func generatePrimitive(typeName string) (interface{}, error) {
	switch typeName {
	case "null":
		return nil, nil
	case "boolean":
		return false, nil
	case "int", "long":
		return 0, nil
	case "float", "double":
		return 0.0, nil
	case "bytes":
		return "", nil
	case "string":
		return "", nil
	default:
		// Named type reference, return empty string as placeholder
		return "", nil
	}
}

func generateUnion(types []interface{}) (interface{}, error) {
	// For unions, prefer the first non-null type
	// If all are null, return null
	for _, t := range types {
		if str, ok := t.(string); ok && str == "null" {
			continue
		}
		return generateValue(t)
	}
	return nil, nil
}

func generateComplex(schema map[string]interface{}) (interface{}, error) {
	schemaType, ok := schema["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' field")
	}

	switch schemaType {
	case "record":
		return generateRecord(schema)
	case "array":
		return generateArray(schema)
	case "map":
		return generateMap(schema)
	case "enum":
		return generateEnum(schema)
	case "fixed":
		return generateFixed(schema)
	default:
		// Primitive type in complex form
		return generatePrimitive(schemaType)
	}
}

func generateRecord(schema map[string]interface{}) (interface{}, error) {
	fields, ok := schema["fields"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("record missing 'fields'")
	}

	result := make(map[string]interface{})

	for _, f := range fields {
		field, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := field["name"].(string)
		if !ok {
			continue
		}

		fieldType, ok := field["type"]
		if !ok {
			continue
		}

		// Check for default value
		if defaultVal, hasDefault := field["default"]; hasDefault {
			result[name] = defaultVal
			continue
		}

		val, err := generateValue(fieldType)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", name, err)
		}
		result[name] = val
	}

	return result, nil
}

func generateArray(schema map[string]interface{}) (interface{}, error) {
	// Return empty array
	return []interface{}{}, nil
}

func generateMap(schema map[string]interface{}) (interface{}, error) {
	// Return empty map
	return map[string]interface{}{}, nil
}

func generateEnum(schema map[string]interface{}) (interface{}, error) {
	symbols, ok := schema["symbols"].([]interface{})
	if !ok || len(symbols) == 0 {
		return "", nil
	}
	// Return first symbol
	if str, ok := symbols[0].(string); ok {
		return str, nil
	}
	return "", nil
}

func generateFixed(schema map[string]interface{}) (interface{}, error) {
	// Return empty string for fixed bytes
	return "", nil
}
