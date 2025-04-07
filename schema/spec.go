package schema

import (
	"bytes"
	"encoding/json"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

type Schema struct {
	SchemaProps `json:",inline"`
	ExtraProps  map[string]any `json:"-"`
	Extensions  map[string]any `json:"-"`
}

type SchemaProps struct {
	Ref                  string                         `json:"$ref,omitempty"`
	Schema               string                         `json:"$schema,omitempty"`
	ID                   string                         `json:"id,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Type                 spec.StringOrArray             `json:"type,omitempty"`
	Nullable             bool                           `json:"nullable,omitempty"`
	Format               string                         `json:"format,omitempty"`
	Title                string                         `json:"title,omitempty"`
	Default              any                            `json:"default,omitempty"`
	Maximum              *float64                       `json:"maximum,omitempty"`
	ExclusiveMaximum     bool                           `json:"exclusiveMaximum,omitempty"`
	Minimum              *float64                       `json:"minimum,omitempty"`
	ExclusiveMinimum     bool                           `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int64                         `json:"maxLength,omitempty"`
	MinLength            *int64                         `json:"minLength,omitempty"`
	Pattern              string                         `json:"pattern,omitempty"`
	MaxItems             *int64                         `json:"maxItems,omitempty"`
	MinItems             *int64                         `json:"minItems,omitempty"`
	UniqueItems          bool                           `json:"uniqueItems,omitempty"`
	MultipleOf           *float64                       `json:"multipleOf,omitempty"`
	Enum                 []any                          `json:"enum,omitempty"`
	MaxProperties        *int64                         `json:"maxProperties,omitempty"`
	MinProperties        *int64                         `json:"minProperties,omitempty"`
	Required             []string                       `json:"required,omitempty"`
	AllOf                []Schema                       `json:"allOf,omitempty"`
	OneOf                []Schema                       `json:"oneOf,omitempty"`
	AnyOf                []Schema                       `json:"anyOf,omitempty"`
	Not                  *Schema                        `json:"not,omitempty"`
	Properties           SchemaProperties               `json:"properties,omitempty"`
	AdditionalProperties *SchemaOrBool                  `json:"additionalProperties,omitempty"`
	PatternProperties    SchemaProperties               `json:"patternProperties,omitempty"`
	Dependencies         map[string]SchemaOrStringArray `json:"dependencies,omitempty"`
	AdditionalItems      *SchemaOrBool                  `json:"additionalItems,omitempty"`
	Definitions          map[string]Schema              `json:"definitions,omitempty"`
	Items                SchemaOrArray                  `json:"items,omitempty"`
	Example              any                            `json:"example,omitempty"`
	Discriminator        string                         `json:"discriminator,omitempty"`
	ReadOnly             bool                           `json:"readOnly,omitempty"`
	XML                  *spec.XMLObject                `json:"xml,omitempty"`
	ExternalDocs         *spec.ExternalDocumentation    `json:"externalDocs,omitempty"`
}

// MarshalJSON marshal this to JSON
func (s Schema) MarshalJSON() ([]byte, error) {
	props, err := json.Marshal(s.SchemaProps)
	if err != nil {
		return nil, err
	}
	extjson, err := json.Marshal(s.Extensions)
	if err != nil {
		return nil, err
	}
	extprops, err := json.Marshal(s.ExtraProps)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(props, extjson, extprops), nil
}

// SchemaProperties is a map representing the properties of a Schema object.
// It knows how to transform its keys into an ordered slice.
type SchemaProperties []SchemaProperty

type SchemaProperty struct {
	Name   string
	Schema Schema
}

// MarshalJSON produces properties as json, keeping their order.
func (properties SchemaProperties) MarshalJSON() ([]byte, error) {
	if properties == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, kv := range properties {
		if i != 0 {
			buf.WriteString(",")
		}
		key, err := json.Marshal(kv.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(": ")
		val, err := json.Marshal(kv.Schema)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// SchemaOrArray represents a value that can either be a Schema
// or an array of Schema. Mainly here for serialization purposes
type SchemaOrArray []Schema

// ContainsType returns true when one of the schemas is of the specified type

// MarshalJSON converts this schema object or array into JSON structure
func (s SchemaOrArray) MarshalJSON() ([]byte, error) {
	if len(s) > 0 {
		return json.Marshal(s[0])
	}
	return json.Marshal(s)
}

// UnmarshalJSON converts this schema object or array from a JSON structure
func (s *SchemaOrArray) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	if first == '{' {
		var sch Schema
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		*s = SchemaOrArray{sch}
	}
	if first == '[' {
		var list []Schema
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*s = SchemaOrArray(list)
	}
	return nil
}

type SchemaOrBool struct {
	Allows bool
	Schema *Schema
}

func (s *SchemaOrBool) MarshalJSON() ([]byte, error) {
	if s.Allows {
		return json.Marshal(true)
	}
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}
	return json.Marshal(false)
}

func (s *SchemaOrBool) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("true")) {
		s.Allows = true
		return nil
	}
	if bytes.Equal(data, []byte("false")) {
		s.Allows = false
		return nil
	}
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	s.Schema = &sch
	return nil
}

// SchemaOrStringArray represents a schema or a string array
type SchemaOrStringArray struct {
	Schema   *Schema
	Property []string
}

// MarshalJSON converts this schema object or array into JSON structure
func (s SchemaOrStringArray) MarshalJSON() ([]byte, error) {
	if len(s.Property) > 0 {
		return json.Marshal(s.Property)
	}
	if s.Schema != nil {
		return json.Marshal(s.Schema)
	}
	return []byte("null"), nil
}

// UnmarshalJSON converts this schema object or array from a JSON structure
func (s *SchemaOrStringArray) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	var nw SchemaOrStringArray
	if first == '{' {
		var sch Schema
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Schema = &sch
	}
	if first == '[' {
		if err := json.Unmarshal(data, &nw.Property); err != nil {
			return err
		}
	}
	*s = nw
	return nil
}
