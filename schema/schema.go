package schema

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/mitchellh/copystructure"
	"gopkg.in/yaml.v3"
)

const DefaultFilePerm = 0o755

var ExtraPropsHandlers = map[string]ExtraPropsHandler{
	"schema":      SchemaOptionHandler,
	"param":       NoopOptionHandler, // ignore @param options
	"hidden":      HiddenOptionHandler,
	"order":       OrderOptionHandler,
	"title":       TitleOptionHandler,
	"x-enum":      XEnumOptionHandler,
	"description": DescriptionOptionHandler,
}

type ExtraPropsHandler func(schema *Schema, sec Section)

type Options struct {
	// parse all schema include not titled
	IncludeAll bool
}

func PurgeSchema(schema *Schema) {
	if schema == nil {
		return
	}
	switch {
	case schema.Type.Contains("object"):
		var purgedProperties SchemaProperties
		for _, item := range schema.Properties {
			if item.Schema.Title == "" {
				continue
			} else {
				PurgeSchema(&item.Schema)
				purgedProperties = append(purgedProperties, item)
			}
		}
		schema.Properties = purgedProperties
	case schema.Type.Contains("array"):
		for i := range schema.Items {
			PurgeSchema(&schema.Items[i])
		}
	}
}

// SplitSchemaI18n
// nolint: gocognit
func SplitSchemaI18n(schema *Schema) map[string]*Schema {
	if schema == nil {
		return nil
	}
	ret := map[string]*Schema{}
	for k, v := range schema.ExtraProps {
		strv, ok := v.(string)
		if !ok {
			continue
		}
		i := strings.IndexRune(k, '.')
		if i < 0 {
			continue
		}
		// this is a dot key,remove from parent
		delete(schema.ExtraProps, k)
		basekey, lang := k[:i], k[i+1:]
		if _, ok := ret[lang]; !ok {
			copyschema := DeepCopySchema(schema)
			removeDotKey(copyschema.ExtraProps)
			ret[lang] = copyschema
		}
		SetSchemaProp(ret[lang], basekey, strv)
	}
	for i, itemschema := range schema.Items {
		for lang, langschema := range SplitSchemaI18n(&itemschema) {
			if _, ok := ret[lang]; !ok {
				ret[lang].Items = slices.Clone(schema.Items)
			}
			ret[lang].Items[i] = *langschema
		}
	}
	for name, val := range schema.Properties {
		for lang, itemlangschema := range SplitSchemaI18n(&val.Schema) {
			if _, ok := ret[lang]; !ok {
				ret[lang] = DeepCopySchema(schema)
			}
			if ret[lang].Properties == nil {
				ret[lang].Properties = SchemaProperties{}
			}
			ret[lang].Properties[name] = SchemaProperty{Name: val.Name, Schema: *itemlangschema}
		}
	}
	return ret
}

func removeDotKey(kvs map[string]any) {
	maps.DeleteFunc(kvs, func(k string, _ any) bool {
		return strings.ContainsRune(k, '.')
	})
}

func DeepCopySchema(in *Schema) *Schema {
	out, err := copystructure.Copy(in)
	if err != nil {
		panic(err)
	}
	// nolint: forcetypeassert
	return out.(*Schema)
}

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
	// update from comment
	completeFromComment(schema, keycomment)
	return schema
}

func completeFromComment(schema *Schema, comment string) {
	annotaionOptions := ParseComment(comment)
	for _, sec := range annotaionOptions {
		if schema.ExtraProps == nil {
			schema.ExtraProps = map[string]any{}
		}
		key := strings.TrimPrefix(sec.Name, "@")
		if handler, ok := ExtraPropsHandlers[key]; ok && handler != nil {
			handler(schema, sec)
		} else {
			DefaultOptionHandler(schema, key, sec)
		}
	}
}

// nolint: gomnd
func ParseComment(comment string) []Section {
	if comment == "" {
		return nil
	}
	sections := []Section{}
	buf := bufio.NewReader(strings.NewReader(comment))
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		if len(line) == 0 {
			continue
		}
		parser := Parser{Data: []byte(line)}

		thissections := parser.ParseSection()
		sections = append(sections, thissections...)
	}
	return sections
}

type Parser struct {
	Data []byte
	Pos  int
}

type Section struct {
	Raw     string
	Name    string
	Value   string
	Options []Option
}

type Option struct {
	Name  string
	Value string
}

// ParseSection parses the section from the comment.
// expr @<scetion> [value] [key=value]...
// example:
// @title test
// @description "a long description"
// @schema format=port;max=65535;min=1
func (p *Parser) ParseSection() []Section {
	for {
		b, ok := p.Read()
		if !ok {
			return nil
		}
		if b != '#' && b != ';' && b != ' ' {
			p.Rewind()
			break
		}
	}
	secname, _, ok := p.ReadIdentity()
	if len(secname) == 0 || !ok {
		return nil
	}
	if secname[0] != '@' {
		return nil
	}
	return p.parseSection(secname)
}

func (p *Parser) parseSection(secname []byte) []Section {
	sec := Section{
		Name: string(secname),
	}
	startpos := p.Pos
	for {
		// example:
		// type=number;format=port;max=65535;min=1
		// foo bar=a,b,c
		// desc="this is a test(abc)." foo=bar
		// @abc2
		// 123 @def 123=1
		key, c, ok := p.ReadIdentity()
		if len(key) == 0 {
			if !ok {
				break
			}
			continue
		}
		if key[0] == '@' {
			// end of this section
			p.Rewind()
			sec.Raw = string(p.Data[startpos:p.Pos])
			return append([]Section{sec}, p.parseSection(key)...)
		}
		if c == '=' {
			val, _, ok := p.ReadIdentity()
			sec.Options = append(sec.Options, Option{Name: string(key), Value: string(val)})
			if !ok {
				break
			}
		} else {
			if len(sec.Options) == 0 {
				sec.Value = string(key)
			} else {
				sec.Options = append(sec.Options, Option{Name: string(key), Value: ""})
			}
		}
	}
	sec.Raw = string(p.Data[startpos:p.Pos])
	return []Section{sec}
}

func (p *Parser) Rewind() {
	p.Pos--
	if p.Pos < 0 {
		p.Pos = 0
	}
}

func (p *Parser) Read() (byte, bool) {
	if p.Pos >= len(p.Data) {
		return 0, false
	}
	b := p.Data[p.Pos]
	p.Pos++
	return b, true
}

func (p *Parser) Until(b byte) ([]byte, bool) {
	start := p.Pos
	for {
		pp, ok := p.Read()
		if !ok {
			return nil, false
		}
		if pp == b {
			return p.Data[start : p.Pos-1], true
		}
	}
}

// ReadIdentity reads a string until it finds a space, comma, or parenthesis.
// example:
//
//	 @abc
//	"foo bar"
//	"foo,bar"
func (p *Parser) ReadIdentity() ([]byte, byte, bool) {
	start := p.Pos
	started := false
	var startch byte
	for {
		b, ok := p.Read()
		if !ok {
			return p.Data[start:p.Pos], 0, false
		}
		switch b {
		case '\'', '"', '`':
			if !started {
				started = true
				startch = b
				start = p.Pos
			} else {
				prevch := p.Data[p.Pos-1]
				// escape char
				if prevch == '\\' {
					continue
				}
				if startch == b {
					return p.Data[start : p.Pos-1], b, true
				}
			}
		case ';', '=':
			if startch == 0 {
				return p.Data[start : p.Pos-1], b, true
			}
		case ' ':
			if started && startch == 0 {
				return p.Data[start : p.Pos-1], b, true
			}
		default:
			if !started {
				started = true
				start = p.Pos - 1
			}
		}
	}
}

// nolint: gomnd
func SchemaOptionHandler(schema *Schema, sec Section) {
	for _, option := range sec.Options {
		SetSchemaProp(schema, option.Name, option.Value)
	}
}

func DefaultOptionHandler(schema *Schema, kind string, sec Section) {
	if sec.Value != "" {
		SetSchemaProp(schema, kind, sec.Value)
		return
	}
	if len(sec.Options) == 0 {
		return
	}
	kvs := map[string]any{}
	for _, option := range sec.Options {
		kvs[option.Name] = formatYamlStr(option.Value)
	}
	if schema.ExtraProps == nil {
		schema.ExtraProps = map[string]any{}
	}
	schema.ExtraProps[kind] = kvs
}

// formatYamlStr convert "true" to bool(true), "123" => int(123)
func formatYamlStr(str string) any {
	if str == "" {
		return nil
	}
	into := map[string]any{}
	if err := yaml.Unmarshal([]byte("key: "+str), &into); err != nil {
		return str
	}
	return into["key"]
}

func NoopOptionHandler(schema *Schema, sec Section) {
}

type HiddenProps struct {
	Operator   HiddenPropsOperator `json:"operator,omitempty"`
	Conditions []HiddenCondition   `json:"conditions,omitempty"`
}

// Ref: https://github.com/kubegems/dashboard/blob/448f9c5767d4232adf4c86b711ae252f5a9e43de/src/views/appstore/components/DeployWizard/Param/index.vue#L160-L185
const (
	HiddenPropsOperatorOr  = "or"
	HiddenPropsOperatorAnd = "and"
	HiddenPropsOperatorNor = "nor"
	HiddenPropsOperatorNot = "not"
)

type HiddenPropsOperator string

type HiddenCondition struct {
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func HiddenOptionHandler(schema *Schema, sec Section) {
	if sec.Value != "" {
		schema.ExtraProps["hidden"] = formatYamlStr(sec.Value)
		return
	}
	options := []Option{}
	var operatorOpetion Option
	for _, option := range sec.Options {
		if option.Name == "operator" {
			operatorOpetion = option
		} else {
			options = append(options, option)
		}
	}
	// convert map k=v to object type {path=jsonpath, value=value}
	operator := operatorOpetion.Value
	if len(options) == 0 {
		return
	}
	if len(options) == 1 {
		// simple type
		for _, option := range options {
			// case : foo!=bar key=foo!
			if strings.HasSuffix(option.Name, "!") {
				schema.ExtraProps["hidden"] = HiddenProps{
					Operator: HiddenPropsOperatorNot,
					Conditions: []HiddenCondition{
						{
							Path:  strings.TrimSuffix(option.Name, "!"),
							Value: formatYamlStr(option.Value),
						},
					},
				}
				return
			}
			schema.ExtraProps["hidden"] = HiddenCondition{Path: option.Name, Value: formatYamlStr(option.Value)}
			return
		}
	}
	if operator == "" {
		operator = HiddenPropsOperatorOr
	}
	props := HiddenProps{
		Operator: HiddenPropsOperator(operator),
	}
	for _, option := range options {
		props.Conditions = append(props.Conditions, HiddenCondition{Path: option.Name, Value: formatYamlStr(option.Value)})
	}
	schema.ExtraProps["hidden"] = props
}

// OrderOptionHandler handle properties order
// Ref: https://github.com/go-openapi/spec/blob/1005cfb91978aa416cfc5a1251b790126390788a/properties.go#L44
func OrderOptionHandler(schema *Schema, sec Section) {
	schema.Extensions["x-order"] = formatYamlStr(sec.Value)
}

func TitleOptionHandler(schema *Schema, sec Section) {
	DefaultOptionHandler(schema, "title", sec)
	// automate add @form=true when @title exists
	if schema.Title != "" {
		SetSchemaProp(schema, "form", "true")
	}
}

// nolint: gomnd,funlen
func SetSchemaProp(schema *Schema, k string, v string) {
	floatPointer := func(in string) *float64 {
		fval, err := strconv.ParseFloat(in, 32)
		if err != nil {
			return nil
		}
		return &fval
	}
	int64Pointer := func(in string) *int64 {
		ival, err := strconv.ParseInt(in, 10, 64)
		if err != nil {
			return nil
		}
		return &ival
	}
	switch k {
	case "min", "minmum":
		schema.Minimum = floatPointer(v)
	case "minLength", "minLen", "minlen":
		schema.MinLength = int64Pointer(v)
	case "maxLength", "maxLen", "maxlen":
		schema.MaxLength = int64Pointer(v)
	case "max", "maxmum":
		schema.Maximum = floatPointer(v)
	case "format":
		schema.Format = v
	case "pattern":
		schema.Pattern = v
	case "required":
		schema.Required = strings.Split(v, ",")
	case "default":
		schema.Default = formatYamlStr(v)
	case "nullable":
		if v == "" {
			schema.Nullable = true
		} else {
			schema.Nullable, _ = strconv.ParseBool(v)
		}
	case "example":
		schema.Example = v
	case "title":
		schema.Title = v
	case "enum":
		enums := []any{}
		for _, item := range strings.Split(v, ",") {
			enums = append(enums, formatYamlStr(item))
		}
		schema.Enum = enums
	case "description":
		schema.Description = v
	case "type":
		if !schema.Type.Contains(v) {
			// use prepend
			if v != "null" {
				schema.Type = append([]string{v}, schema.Type...)
			} else {
				schema.Type = append(schema.Type, v)
			}
		}
	case "items":
		items := SchemaOrArray{}
		if err := json.Unmarshal([]byte(v), &items); err != nil {
			fmt.Printf("Unable decode schema items values: %s error: %s\n", v, err.Error())
		}
		schema.Items = items
	default:
		if schema.ExtraProps == nil {
			schema.ExtraProps = map[string]any{}
		}
		schema.ExtraProps[k] = formatExtraValue(v)
	}
}

func formatExtraValue(v string) any {
	if v == "" {
		return nil
	}
	if v[0] == '{' || v[0] == '[' {
		var obj any
		if err := json.Unmarshal([]byte(v), &obj); err != nil {
			return v
		}
		return obj
	}
	return formatYamlStr(v)
}

func XEnumOptionHandler(schema *Schema, sec Section) {
	if schema.ExtraProps == nil {
		schema.ExtraProps = map[string]any{}
	}
	xenum := []map[string]any{}
	for _, option := range sec.Options {
		xenum = append(xenum, map[string]any{"text": option.Value, "value": formatYamlStr(option.Name)})
	}
	if _, ok := schema.ExtraProps["render"]; !ok {
		schema.ExtraProps["render"] = "radio"
	}
	schema.ExtraProps["x-enum"] = xenum
}

func DescriptionOptionHandler(schema *Schema, sec Section) {
	schema.Description = sec.Raw
}
