package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"
	"xiaoshiai.cn/schema/schema"
)

const DefaultFilePerm = 0o755

func main() {
	cmd := NewCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.MinimumNArgs(1),
		Long: `
		Example:
		helm-schema ./charts/mychart
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			go func() {
				<-ctx.Done()
				os.Exit(1)
			}()
			for _, arg := range args {
				if err := Generate(arg, schema.Options{}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

func Generate(glob string, options schema.Options) error {
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	for _, path := range matches {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			continue
		}
		if err := GenerateWriteSchema(path, options); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	}
	return nil
}

func GenerateWriteSchema(chartpath string, options schema.Options) error {
	if filepath.Base(chartpath) == "values.yaml" {
		chartpath = filepath.Dir(chartpath)
	}
	valuesfile := filepath.Join(chartpath, "values.yaml")
	fmt.Printf("Reading %s\n", valuesfile)
	valuecontent, err := os.ReadFile(valuesfile)
	if err != nil {
		return err
	}
	item, err := schema.GenerateSchema(valuecontent)
	if err != nil {
		return err
	}
	i18nschemas, err := schema.CompleteI18n(*item)
	if err != nil {
		return err
	}
	if options.I18nDirectory != "" {
		if err := os.MkdirAll(filepath.Join(chartpath, options.I18nDirectory), DefaultFilePerm); err != nil {
			return err
		}
	}
	for lang, langschema := range i18nschemas.Locales {
		filename := filepath.Join(chartpath, options.I18nDirectory, fmt.Sprintf("values.schema.%s.json", lang))
		if !options.IncludeAll {
			schema.PurgeSchema(item)
		}
		if langschema.Empty() {
			fmt.Printf("Empty schema of i18n %s schema\n", lang)
			return nil
		}
		if err := WriteJson(filename, langschema); err != nil {
			return err
		}
	}
	if !options.IncludeAll {
		schema.PurgeSchema(item)
	}
	if i18nschemas.Orignal.Empty() {
		fmt.Printf("Empty schema")
		return nil
	}
	return WriteJson(filepath.Join(chartpath, "values.schema.json"), i18nschemas.Orignal)
}

func WriteJson(filename string, data any) error {
	schemacontent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), DefaultFilePerm); err != nil {
		return err
	}
	fmt.Printf("Writing %s\n", filename)
	return os.WriteFile(filename, schemacontent, DefaultFilePerm)
}
