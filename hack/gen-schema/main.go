// Command gen-schema emits a JSON Schema for the Yamlfile v1alpha1 types.
// It is intended to be run at documentation generation time (see Makefile)
// so that docs/static/schema/v1alpha1.json is always produced from the exact
// Go structs in pkg/spec/v1alpha1 at that moment.
//
// Usage: go run ./hack/gen-schema -o docs/static/schema/v1alpha1.json
//
// The generator uses only the standard library + the local spec package.
// It specially handles the Step discriminated union (producing oneOf) and
// marks Extension maps (yaml:",inline") as additionalProperties: true for
// the forward-compat model used throughout the project.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	spec "github.com/builderhub/yamlfile/pkg/spec/v1alpha1"
)

func main() {
	outPath := flag.String("o", "", "output path for the generated schema (required)")
	flag.Parse()
	if *outPath == "" {
		fmt.Fprintln(os.Stderr, "error: -o <path> is required")
		os.Exit(2)
	}

	schema := buildSchema()

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal schema: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, append(data, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *outPath)
}

// rootSchema is the shape we emit for the top level.
type rootSchema map[string]interface{}

func buildSchema() rootSchema {
	// Collect reusable definitions first.
	defs := map[string]interface{}{}

	// Order matters only for human readability of the output.
	defs["Yamlfile"] = schemaFor(reflect.TypeOf(spec.Yamlfile{}), defs)
	defs["Defaults"] = schemaFor(reflect.TypeOf(spec.Defaults{}), defs)
	defs["TargetSpec"] = schemaFor(reflect.TypeOf(spec.TargetSpec{}), defs)
	defs["BuildRef"] = schemaFor(reflect.TypeOf(spec.BuildRef{}), defs)
	defs["Step"] = stepDiscriminatedUnion(defs) // special: oneOf
	defs["RunSpec"] = schemaFor(reflect.TypeOf(spec.RunSpec{}), defs)
	defs["CopySpec"] = schemaFor(reflect.TypeOf(spec.CopySpec{}), defs)
	defs["EnvSpec"] = schemaFor(reflect.TypeOf(spec.EnvSpec{}), defs)
	defs["ArgSpec"] = schemaFor(reflect.TypeOf(spec.ArgSpec{}), defs)
	defs["WorkdirSpec"] = schemaFor(reflect.TypeOf(spec.WorkdirSpec{}), defs)
	defs["SecretMount"] = schemaFor(reflect.TypeOf(spec.SecretMount{}), defs)

	root := rootSchema{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://builderhub.github.io/Yamlfile/schema/v1alpha1.json",
		"title":       "Yamlfile v1alpha1",
		"description": "BuildKit frontend (v1alpha1) declarative build definition. The schema is generated from the Go types in pkg/spec/v1alpha1 and is the source of truth for editors and validation.",
		"$defs":       defs,
		"$ref":        "#/$defs/Yamlfile",
	}
	return root
}

// stepDiscriminatedUnion builds the special oneOf shape for Step so that exactly
// one of run/copy/env/arg/workdir is present (matching the parser validation).
func stepDiscriminatedUnion(_ map[string]interface{}) map[string]interface{} {
	arms := []map[string]interface{}{
		{"required": []string{"run"}, "properties": map[string]interface{}{"run": map[string]interface{}{"$ref": "#/$defs/RunSpec"}}},
		{"required": []string{"copy"}, "properties": map[string]interface{}{"copy": map[string]interface{}{"$ref": "#/$defs/CopySpec"}}},
		{"required": []string{"env"}, "properties": map[string]interface{}{"env": map[string]interface{}{"$ref": "#/$defs/EnvSpec"}}},
		{"required": []string{"arg"}, "properties": map[string]interface{}{"arg": map[string]interface{}{"$ref": "#/$defs/ArgSpec"}}},
		{"required": []string{"workdir"}, "properties": map[string]interface{}{"workdir": map[string]interface{}{"$ref": "#/$defs/WorkdirSpec"}}},
	}
	// Also allow the extension fields at the step level (forward compat).
	for i := range arms {
		arms[i]["additionalProperties"] = true
	}
	return map[string]interface{}{
		"type":  "object",
		"oneOf": arms,
		// A step may also carry unknown keys (future kinds or extensions captured by the parser).
		"additionalProperties": true,
	}
}

// schemaFor recursively produces a JSON Schema fragment for t, populating defs
// for named struct types encountered.
func schemaFor(t reflect.Type, defs map[string]interface{}) map[string]interface{} {
	// Dereference pointers for the schema shape (presence is controlled by omitempty / required lists).
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": schemaFor(t.Elem(), defs),
		}
	case reflect.Map:
		// We treat map[string]T as object with additionalProperties.
		valSchema := schemaFor(t.Elem(), defs)
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": valSchema,
		}
	case reflect.Struct:
		// Named structs get entered into $defs when we see them at the top level in buildSchema.
		// For inline/anonymous or already-seen, we still build the object shape.
		props := map[string]interface{}{}
		required := []string{}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name, omitempty, inline := yamlName(f)
			if name == "" || name == "-" {
				continue
			}

			fieldSchema := schemaFor(f.Type, defs)

			// Attach a small description from the godoc on the field (first sentence-ish).
			if doc := fieldDoc(f); doc != "" {
				fieldSchema["description"] = doc
			}

			props[name] = fieldSchema

			if !omitempty && !inline {
				// Only mark as required when the yaml tag did not have omitempty and it is not an inline extension map.
				// This keeps the schema practical for YAML users while still being useful.
				required = append(required, name)
			}

			// inline (e.g. our Extensions maps) is handled at the struct level via hasInlineExtension + additionalProperties.
		}

		sch := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			sch["required"] = required
		}

		// If this struct has any inline extension map (or conventionally named Extensions), be permissive.
		if hasInlineExtension(t) {
			sch["additionalProperties"] = true
		}

		return sch

	default:
		// Fallback: any
		return map[string]interface{}{}
	}
}

func yamlName(f reflect.StructField) (name string, omitempty, inline bool) {
	tag := f.Tag.Get("yaml")
	if tag == "" {
		// Fall back to lowercase field name (not common in this project).
		return strings.ToLower(f.Name[:1]) + f.Name[1:], false, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		switch strings.TrimSpace(p) {
		case "omitempty":
			omitempty = true
		case "inline":
			inline = true
		}
	}
	if name == "" {
		name = strings.ToLower(f.Name[:1]) + f.Name[1:]
	}
	return name, omitempty, inline
}

func hasInlineExtension(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		_, _, inline := yamlName(f)
		if inline {
			return true
		}
		if strings.EqualFold(f.Name, "Extensions") {
			return true
		}
	}
	return false
}

func fieldDoc(_ reflect.StructField) string {
	// reflect.StructField has no direct doc; we keep it simple.
	// A fuller version could parse the package with go/ast, but that is heavy for a docs generator.
	// Descriptions in the emitted schema come primarily from the surrounding prose in syntax-reference.md.
	return ""
}
