package schema

import (
	"github.com/go-openapi/spec"
	"gopkg.in/yaml.v3"
)

func GenerateSchema(values []byte) (*Schema, error) {
	node := &yaml.Node{}
	if err := yaml.Unmarshal(values, node); err != nil {
		return nil, err
	}
	return nodeSchema(node, ""), nil
}

// nolint: funlen
func nodeSchema(node *yaml.Node, keycomment string) *Schema {
	schema := &Schema{}
	switch node.Kind {
	case yaml.DocumentNode:
		rootschema := nodeSchema(node.Content[0], "")
		if rootschema == nil {
			return nil
		}
		rootschema.Schema = "http://json-schema.org/schema#"
		return rootschema
	case yaml.MappingNode:
		schema.Type = spec.StringOrArray{"object"}
		if schema.Properties == nil {
			schema.Properties = SchemaProperties{}
		}
		for i := 0; i < len(node.Content); i += 2 {
			key, keycomment := node.Content[i].Value, node.Content[i].HeadComment
			objectProperty := nodeSchema(node.Content[i+1], keycomment)
			if objectProperty == nil {
				continue
			}
			schema.Properties = append(schema.Properties, SchemaProperty{Name: key, Schema: *objectProperty})
		}
	case yaml.SequenceNode:
		schema.Type = spec.StringOrArray{"array"}
		var schemas []Schema
		for _, itemnode := range node.Content {
			itemProperty := nodeSchema(itemnode, "")
			if itemProperty == nil {
				continue
			}
			schemas = append(schemas, *itemProperty)
		}
		if len(schemas) == 1 {
			schema.Items = SchemaOrArray{schemas[0]}
		} else {
			schema.Items = SchemaOrArray(schemas)
		}
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str", "!binary":
			schema.Type = spec.StringOrArray{"string"}
		case "!!int":
			schema.Type = spec.StringOrArray{"integer"}
		case "!!float":
			schema.Type = spec.StringOrArray{"number"}
		case "!!bool":
			schema.Type = spec.StringOrArray{"boolean"}
		case "!!timestamp":
			schema.Type = spec.StringOrArray{"string"}
			schema.Format = "data-time"
		case "!!null":
			schema.Type = spec.StringOrArray{"null"}
		default:
			schema.Type = spec.StringOrArray{"object"}
		}
		// set default value
		if node.Value != "" {
			if schema.Type.Contains("string") {
				schema.Default = node.Value // string type's default values is string
			} else {
				schema.Default = formatYamlStr(node.Value)
			}
		}
	}
	schema.Comment = keycomment
	return schema
}
