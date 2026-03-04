// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// patchspec sanitises a Qlik OpenAPI JSON spec so that oapi-codegen can parse
// it correctly. It fixes cases where a URL template contains a {param}
// placeholder that is not declared as an "in: path" parameter – by removing
// those dangling placeholders from the path key and re-keying the operation.
//
// Usage:
//
//	go run ./cmd/patchspec <input.json> <output.json>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var placeholderRe = regexp.MustCompile(`\{(\w+)\}`)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: patchspec <input.json> <output.json>")
		os.Exit(1)
	}
	inputPath := os.Args[1]
	outputPath := os.Args[2]

	raw, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}

	var spec map[string]any
	if err := json.Unmarshal(raw, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		writeJSON(spec, outputPath)
		return
	}

	patched := make(map[string]any, len(paths))
	for rawPath, item := range paths {
		newPath := fixPath(rawPath, item)
		if newPath != rawPath {
			fmt.Fprintf(os.Stderr, "patched path: %q → %q\n", rawPath, newPath)
		}
		patched[newPath] = item
	}
	spec["paths"] = patched

	// Remove x-go-type entries that are objects (maps) instead of strings.
	// oapi-codegen expects x-go-type to be a plain string type name.
	// Object-valued x-go-type entries typically reference private/internal packages
	// that are not available, so we strip them and let oapi-codegen derive the type.
	fixXGoType(spec)

	// Ensure no two component schemas produce the same Go type name after PascalCasing.
	fixDuplicateSchemaNames(spec)

	// Remove format: date-time from fields the API documents as returning an empty
	// string when unset. An empty string is not a valid date-time, so keeping
	// format: date-time causes json.Unmarshal to fail at runtime.
	fixEmptyableTimeFields(spec)

	writeJSON(spec, outputPath)
}

// fixPath strips placeholder segments from rawPath that are not backed by an
// "in: path" parameter declaration in any of the operations on that path item.
func fixPath(rawPath string, item any) string {
	ops, ok := item.(map[string]any)
	if !ok {
		return rawPath
	}

	// Collect all path param names declared across every operation on this item.
	declared := map[string]bool{}
	for _, method := range []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"} {
		op, ok := ops[method].(map[string]any)
		if !ok {
			continue
		}
		for _, p := range asSlice(op["parameters"]) {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if pm["in"] == "path" {
				if name, ok := pm["name"].(string); ok {
					declared[name] = true
				}
			}
		}
	}

	// Remove any {placeholder} in the URL that has no matching path param.
	result := placeholderRe.ReplaceAllStringFunc(rawPath, func(m string) string {
		name := m[1 : len(m)-1] // strip { }
		if !declared[name] {
			return "" // drop it
		}
		return m
	})

	// Clean up double slashes that can appear after dropping a segment.
	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}
	result = strings.TrimRight(result, "/")

	return result
}

// toPascalCase converts a schema key to a Go-exported identifier, matching
// the heuristic oapi-codegen uses internally.
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// fixDuplicateSchemaNames detects component schemas whose names would collide
// after PascalCasing and adds an x-go-name extension to the "lesser" one
// (lower-case original key) to give it a unique name.
func fixDuplicateSchemaNames(spec map[string]any) {
	components, ok := spec["components"].(map[string]any)
	if !ok {
		return
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return
	}

	// Map from pascalCase name → first schema key that claimed it.
	claimed := map[string]string{}
	for key := range schemas {
		pascal := toPascalCase(key)
		if existing, clash := claimed[pascal]; clash {
			// key collides with existing; add x-go-name to the lowercase/later one.
			loser := key
			if existing < key {
				loser = key
			} else {
				loser = existing
			}
			schema, ok := schemas[loser].(map[string]any)
			if !ok {
				continue
			}
			if _, already := schema["x-go-name"]; already {
				continue
			}
			// Use the original key as the basis, capitalised + "Value" suffix.
			newName := toPascalCase(loser) + "Value"
			schema["x-go-name"] = newName
			fmt.Fprintf(os.Stderr, "renamed duplicate schema %q → x-go-name %q\n", loser, newName)
		} else {
			claimed[pascal] = key
		}
	}
}

// fixXGoType recursively walks the spec and removes any "x-go-type" field
// whose value is a map (object) rather than a plain string.
func fixXGoType(v any) {
	switch node := v.(type) {
	case map[string]any:
		if xgt, ok := node["x-go-type"]; ok {
			if _, isMap := xgt.(map[string]any); isMap {
				fmt.Fprintf(os.Stderr, "removed object-valued x-go-type\n")
				delete(node, "x-go-type")
			}
		}
		for _, child := range node {
			fixXGoType(child)
		}
	case []any:
		for _, child := range node {
			fixXGoType(child)
		}
	}
}

// fixEmptyableTimeFields removes the "format": "date-time" annotation from
// schema properties whose description documents that the API may return an
// empty string for that field (e.g. publishTime when the app is unpublished).
// Without this fix oapi-codegen generates *time.Time and json.Unmarshal rejects
// the empty string at runtime.
func fixEmptyableTimeFields(spec map[string]any) {
	// Fields that the Qlik API may return as "" instead of a proper RFC3339 value.
	emptyable := map[string]bool{
		"publishTime":    true,
		"publishedAt":    true, // "empty if unpublished"
		"lastReloadTime": true, // empty before first reload
	}

	components, ok := spec["components"].(map[string]any)
	if !ok {
		return
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return
	}

	for schemaName, schema := range schemas {
		schemaMap, ok := schema.(map[string]any)
		if !ok {
			continue
		}
		properties, ok := schemaMap["properties"].(map[string]any)
		if !ok {
			continue
		}
		for fieldName, field := range properties {
			if !emptyable[fieldName] {
				continue
			}
			fieldMap, ok := field.(map[string]any)
			if !ok {
				continue
			}
			if fmt_, ok := fieldMap["format"]; ok && fmt_ == "date-time" {
				delete(fieldMap, "format")
				fmt.Fprintf(os.Stderr, "removed date-time format from %s.%s (field accepts empty string)\n", schemaName, fieldName)
			}
		}
	}
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func writeJSON(v any, path string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote patched spec → %s\n", path)
}
