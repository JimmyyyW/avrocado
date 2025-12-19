package avro

import (
	"encoding/json"
	"fmt"
)

// templateGenerator holds state while generating a template,
// including a registry of named types encountered during parsing.
type templateGenerator struct {
	namedTypes map[string]map[string]interface{}
}

// GenerateTemplate creates a JSON template from an Avro schema.
// The template contains placeholder values for each field.
func GenerateTemplate(schemaJSON string) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return "", fmt.Errorf("parsing schema: %w", err)
	}

	gen := &templateGenerator{
		namedTypes: make(map[string]map[string]interface{}),
	}

	// First pass: collect all named types
	gen.collectNamedTypes(schema)

	// Second pass: generate the template
	result, err := gen.generateValue(schema)
	if err != nil {
		return "", err
	}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatting template: %w", err)
	}

	return string(pretty), nil
}

// collectNamedTypes recursively finds and registers all named types in the schema
func (g *templateGenerator) collectNamedTypes(schema interface{}) {
	switch s := schema.(type) {
	case map[string]interface{}:
		// Check if this is a named type (record, enum, fixed)
		if typeName, ok := s["type"].(string); ok {
			switch typeName {
			case "record", "enum", "fixed":
				if name, ok := s["name"].(string); ok {
					// Register with full name if namespace exists
					fullName := name
					if ns, ok := s["namespace"].(string); ok {
						fullName = ns + "." + name
					}
					g.namedTypes[name] = s
					g.namedTypes[fullName] = s
				}
			}
		}

		// Recurse into fields
		if fields, ok := s["fields"].([]interface{}); ok {
			for _, f := range fields {
				if field, ok := f.(map[string]interface{}); ok {
					if fieldType, ok := field["type"]; ok {
						g.collectNamedTypes(fieldType)
					}
				}
			}
		}

		// Recurse into array items
		if items, ok := s["items"]; ok {
			g.collectNamedTypes(items)
		}

		// Recurse into map values
		if values, ok := s["values"]; ok {
			g.collectNamedTypes(values)
		}

	case []interface{}:
		// Union type - recurse into each option
		for _, t := range s {
			g.collectNamedTypes(t)
		}
	}
}

func (g *templateGenerator) generateValue(schema interface{}) (interface{}, error) {
	switch s := schema.(type) {
	case string:
		return g.generatePrimitive(s)
	case []interface{}:
		return g.generateUnion(s)
	case map[string]interface{}:
		return g.generateComplex(s)
	default:
		return nil, fmt.Errorf("unexpected schema type: %T", schema)
	}
}

func (g *templateGenerator) generatePrimitive(typeName string) (interface{}, error) {
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
		// Named type reference - look it up
		if namedType, ok := g.namedTypes[typeName]; ok {
			return g.generateComplex(namedType)
		}
		// Unknown type, return empty string
		return "", nil
	}
}

func (g *templateGenerator) generateUnion(types []interface{}) (interface{}, error) {
	// For unions, prefer the first non-null type
	// If all are null, return null
	for _, t := range types {
		if str, ok := t.(string); ok && str == "null" {
			continue
		}
		return g.generateValue(t)
	}
	return nil, nil
}

func (g *templateGenerator) generateComplex(schema map[string]interface{}) (interface{}, error) {
	schemaType, ok := schema["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' field")
	}

	switch schemaType {
	case "record":
		return g.generateRecord(schema)
	case "array":
		return g.generateArray(schema)
	case "map":
		return g.generateMap(schema)
	case "enum":
		return g.generateEnum(schema)
	case "fixed":
		return g.generateFixed(schema)
	default:
		// Primitive type in complex form
		return g.generatePrimitive(schemaType)
	}
}

func (g *templateGenerator) generateRecord(schema map[string]interface{}) (interface{}, error) {
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

		val, err := g.generateValue(fieldType)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", name, err)
		}
		result[name] = val
	}

	return result, nil
}

func (g *templateGenerator) generateArray(schema map[string]interface{}) (interface{}, error) {
	// Return empty array
	return []interface{}{}, nil
}

func (g *templateGenerator) generateMap(schema map[string]interface{}) (interface{}, error) {
	// Return empty map
	return map[string]interface{}{}, nil
}

func (g *templateGenerator) generateEnum(schema map[string]interface{}) (interface{}, error) {
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

func (g *templateGenerator) generateFixed(schema map[string]interface{}) (interface{}, error) {
	// Return empty string for fixed bytes
	return "", nil
}
