package avro

import (
	"encoding/json"
	"fmt"

	"github.com/linkedin/goavro/v2"
)

// Validator validates JSON data against an Avro schema.
type Validator struct {
	codec *goavro.Codec
}

// NewValidator creates a new Avro validator from a schema JSON string.
func NewValidator(schemaJSON string) (*Validator, error) {
	codec, err := goavro.NewCodec(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	return &Validator{codec: codec}, nil
}

// Validate checks if the JSON data is valid according to the schema.
// Returns nil if valid, or an error describing the validation failure.
func (v *Validator) Validate(jsonData string) error {
	// Parse JSON to native Go types
	var native interface{}
	if err := json.Unmarshal([]byte(jsonData), &native); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Convert to Avro-compatible format and validate by encoding
	_, err := v.codec.BinaryFromNative(nil, native)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// Encode converts JSON data to Avro binary format.
// Returns the binary data or an error if validation fails.
func (v *Validator) Encode(jsonData string) ([]byte, error) {
	var native interface{}
	if err := json.Unmarshal([]byte(jsonData), &native); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	binary, err := v.codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	return binary, nil
}

// ValidateAndEncode validates JSON data and returns Avro binary if valid.
func ValidateAndEncode(schemaJSON, jsonData string) ([]byte, error) {
	v, err := NewValidator(schemaJSON)
	if err != nil {
		return nil, err
	}
	return v.Encode(jsonData)
}
