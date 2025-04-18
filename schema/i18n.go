package schema

import (
	"fmt"

	"github.com/mitchellh/copystructure"
)

type Options struct {
	// parse all schema include not titled
	IncludeAll    bool
	I18nDirectory string
}

func PurgeSchema(schema *Schema) {
	if schema == nil {
		return
	}
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
	for i := range schema.Items {
		PurgeSchema(&schema.Items[i])
	}
}

func CompleteI18n(schema Schema) (*I18nSchema, error) {
	i18nschemas, err := CompleteI18nFromComment(schema, schema.Comment)
	if err != nil {
		return nil, err
	}

	items := make([]I18nSchema, len(i18nschemas.Orignal.Items))
	for i := range i18nschemas.Orignal.Items {
		subi18nschemas, err := CompleteI18n(i18nschemas.Orignal.Items[i])
		if err != nil {
			return nil, err
		}
		items[i] = *subi18nschemas
	}
	properties := make([]I18nSchema, len(i18nschemas.Orignal.Properties))
	for i, item := range i18nschemas.Orignal.Properties {
		subi18nschemas, err := CompleteI18n(item.Schema)
		if err != nil {
			return nil, err
		}
		properties[i] = *subi18nschemas
	}

	// must make  i18nschemas.Orignal's properties updated fully before deepcopy it.
	for i, subi18nschemas := range items {
		i18nschemas.Orignal.Items[i] = *subi18nschemas.Orignal
	}
	for i, subi18nschemas := range properties {
		i18nschemas.Orignal.Properties[i].Schema = *subi18nschemas.Orignal
	}

	for i, subi18nschemas := range items {
		for locale, subi18nschema := range subi18nschemas.Locales {
			if _, ok := i18nschemas.Locales[locale]; !ok {
				i18nschemas.Locales[locale] = DeepCopySchema(i18nschemas.Orignal)
			}
			i18nschemas.Locales[locale].Items[i] = *subi18nschema
		}
	}
	for i, subi18nschemas := range properties {
		for locale, subi18nschema := range subi18nschemas.Locales {
			if _, ok := i18nschemas.Locales[locale]; !ok {
				i18nschemas.Locales[locale] = DeepCopySchema(i18nschemas.Orignal)
			}
			i18nschemas.Locales[locale].Properties[i].Schema = *subi18nschema
		}
	}
	return i18nschemas, nil
}

type I18nSchema struct {
	Orignal *Schema
	Locales map[string]*Schema
}

func CompleteI18nFromComment(schema Schema, comment string) (*I18nSchema, error) {
	nolocale := []Section{}
	bylocale := map[string][]Section{}
	for _, sec := range ParseComment(comment) {
		key, locale := SplitKeyLocale(sec.Name)
		if locale != "" {
			sec.Name = key // reset section name to no locate
			bylocale[locale] = append(bylocale[key], sec)
		} else {
			nolocale = append(nolocale, sec)
		}
	}

	var errs []error
	nolocaleschema := DeepCopySchema(&schema)
	for _, sec := range nolocale {
		if err := CompleteFromCommentSection(nolocaleschema, sec); err != nil {
			errs = append(errs, err)
		}
	}
	localeschemas := map[string]*Schema{}
	for locale, secs := range bylocale {
		localeSchema := DeepCopySchema(nolocaleschema)
		for _, sec := range secs {
			if err := CompleteFromCommentSection(localeSchema, sec); err != nil {
				errs = append(errs, err)
			}
		}
		localeschemas[locale] = localeSchema
	}
	return &I18nSchema{Orignal: nolocaleschema, Locales: localeschemas}, CombineErrors(errs...)
}

func DeepCopySchema(in *Schema) *Schema {
	out, err := copystructure.Copy(in)
	if err != nil {
		panic(err)
	}
	// nolint: forcetypeassert
	return out.(*Schema)
}

func CombineErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%v", errs)
}
