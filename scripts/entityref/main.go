// Command entityref renders visitor entity pages from JSON schemas plus
// docs/_data/entity-reference.yml. Schema owns fields, enums, filename, and
// sections. The YAML overlay owns voice. Plan is locked and is not emitted.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const overlayRelativePath = "docs/_data/entity-reference.yml"

type Overlay struct {
	Entities []Entity `yaml:"entities"`
}

type Entity struct {
	ID             string            `yaml:"id"`
	Output         string            `yaml:"output"`
	Permalink      string            `yaml:"permalink"`
	Title          string            `yaml:"title"`
	Schema         string            `yaml:"schema"`
	SchemaLabel    string            `yaml:"schema_label"`
	FileDir        string            `yaml:"file_dir"`
	FilePattern    string            `yaml:"file_pattern"`
	HeroLede       string            `yaml:"hero_lede"`
	Format         string            `yaml:"format"`
	CreatedFrom    string            `yaml:"created_from"`
	Sources        []Source          `yaml:"sources"`
	StatusIntro    string            `yaml:"status_intro"`
	StatusPath     string            `yaml:"status_path"`
	StatusKinds    map[string]string `yaml:"status_kinds"`
	StatusMeanings map[string]string `yaml:"status_meanings"`
	Fields         []FieldSpec       `yaml:"fields"`
	ExtraIllegal   []string          `yaml:"extra_illegal"`
	Constraints    []string          `yaml:"constraints"`
	Validate       string            `yaml:"validate"`
	Reviewer       Reviewer          `yaml:"reviewer"`
	Notes          []string          `yaml:"notes"`
	Also           []Link            `yaml:"also"`
}

type Source struct {
	Work string `yaml:"work"`
	Path string `yaml:"path"`
}

type FieldSpec struct {
	Key      string `yaml:"key"`
	Required string `yaml:"required"`
	Form     string `yaml:"form"`
}

type Reviewer struct {
	Name    string   `yaml:"name"`
	Intro   string   `yaml:"intro"`
	Bullets []string `yaml:"bullets"`
}

type Link struct {
	Href  string `yaml:"href"`
	Label string `yaml:"label"`
}

func main() {
	root, err := findRepoRoot(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "entityref: %v\n", err)
		os.Exit(1)
	}
	check := len(os.Args) > 1 && os.Args[1] == "-check"
	if err := Run(root, check, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "entityref: %v\n", err)
		os.Exit(1)
	}
}

func findRepoRoot(start string) (string, error) {
	wd, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, overlayRelativePath)); err == nil {
			return d, nil
		}
	}
	return "", fmt.Errorf("repository root not found from %s (missing %s)", wd, overlayRelativePath)
}

func Run(root string, check bool, w io.Writer) error {
	overlayPath := filepath.Join(root, overlayRelativePath)
	raw, err := os.ReadFile(overlayPath)
	if err != nil {
		return fmt.Errorf("read overlay: %w", err)
	}
	var overlay Overlay
	if err := yaml.Unmarshal(raw, &overlay); err != nil {
		return fmt.Errorf("parse overlay: %w", err)
	}
	for _, ent := range overlay.Entities {
		if ent.ID == "plan" {
			fmt.Fprintf(os.Stderr, "entityref: skip locked %s\n", ent.ID)
			continue
		}
		page, err := render(root, ent)
		if err != nil {
			return fmt.Errorf("%s: %w", ent.ID, err)
		}
		out := filepath.Join(root, ent.Output)
		if check {
			old, err := os.ReadFile(out)
			if err != nil {
				return fmt.Errorf("%s: %w", ent.Output, err)
			}
			if string(old) != page {
				return fmt.Errorf("%s drifted from schema+overlay; run go run ./scripts/entityref", ent.Output)
			}
			if _, err := fmt.Fprintf(w, "entityref: ok %s\n", ent.Output); err != nil {
				return fmt.Errorf("write status: %w", err)
			}
			continue
		}
		if err := os.WriteFile(out, []byte(page), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", ent.Output, err)
		}
		if _, err := fmt.Fprintf(w, "entityref: wrote %s from %s\n", ent.Output, schemaSource(ent)); err != nil {
			return fmt.Errorf("write status: %w", err)
		}
	}
	return nil
}

func schemaSource(ent Entity) string {
	if ent.Schema == "" {
		return "overlay"
	}
	return ent.Schema
}

func render(root string, ent Entity) (string, error) {
	var schema map[string]any
	if ent.Schema != "" {
		b, err := os.ReadFile(filepath.Join(root, ent.Schema))
		if err != nil {
			return "", fmt.Errorf("read schema %s: %w", ent.Schema, err)
		}
		if err := json.Unmarshal(b, &schema); err != nil {
			return "", fmt.Errorf("parse schema %s: %w", ent.Schema, err)
		}
		schema = mergeBase(root, schema)
	} else {
		schema = map[string]any{}
	}

	schemaLabel := ent.SchemaLabel
	if schemaLabel == "" {
		schemaLabel = schemaVersion(schema, ent)
	}
	fileLine := ent.FilePattern
	if fileLine == "" {
		fileLine = joinFile(ent.FileDir, humanizePattern(asString(schema["filename_pattern"])))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "title: %s\n", ent.Title)
	fmt.Fprintf(&b, "layout: default\n")
	fmt.Fprintf(&b, "permalink: %s\n", ent.Permalink)
	fmt.Fprintf(&b, "page_kind: entity\n")
	fmt.Fprintf(&b, "hero_question: %q\n", ent.Title)
	fmt.Fprintf(&b, "hero_lede: %q\n", ent.HeroLede)
	fmt.Fprintf(&b, "---\n\n")
	fmt.Fprintf(&b, "<!-- generated by go run ./scripts/entityref from %s + docs/_data/entity-reference.yml. do not hand-edit. -->\n\n", schemaSource(ent))

	fmt.Fprintf(&b, "<dl class=\"entity-meta\">\n")
	kind := "Artifact"
	if ent.ID == "pack" {
		kind = "Pack"
	}
	fmt.Fprintf(&b, "<dt>Kind</dt><dd>%s</dd>\n", kind)
	fmt.Fprintf(&b, "<dt>Schema</dt><dd><code>%s</code></dd>\n", escape(schemaLabel))
	fmt.Fprintf(&b, "<dt>File</dt><dd><code>%s</code></dd>\n", escape(fileLine))
	fmt.Fprintf(&b, "<dt>Format</dt><dd>%s</dd>\n", mdCode(ent.Format))
	fmt.Fprintf(&b, "<dt>Created from</dt><dd>%s</dd>\n", mdCode(ent.CreatedFrom))
	fmt.Fprintf(&b, "</dl>\n")

	page := b.String()

	var rest strings.Builder
	if len(ent.Sources) > 0 {
		rest.WriteString("\n## Work path\n\n")
		rest.WriteString(sourcesTable(ent.Sources))
	}

	if len(ent.Fields) > 0 {
		rest.WriteString("\n## Fields\n\n")
		rest.WriteString(fieldsTable(schema, ent.Fields))
	}

	if ent.StatusPath != "" || len(ent.StatusMeanings) > 0 {
		values := statusValues(schema, ent.StatusPath)
		if len(values) == 0 {
			values = append(values, sortedKeys(ent.StatusMeanings)...)
		}
		if len(values) > 0 {
			rest.WriteString("\n## Status\n\n")
			if ent.StatusIntro != "" {
				rest.WriteString(ent.StatusIntro + "\n\n")
			}
			rest.WriteString(statusTable(values, ent.StatusKinds, ent.StatusMeanings))
			rest.WriteString(illegalBlock(values, ent))
		}
	}

	if sections := sectionTable(schema); sections != "" {
		rest.WriteString("\n## Sections\n\n")
		rest.WriteString(sections)
	}

	if len(ent.Constraints) > 0 {
		rest.WriteString("\n## Constraints\n\n")
		for _, c := range ent.Constraints {
			fmt.Fprintf(&rest, "- %s\n", c)
		}
	}

	if ent.Validate != "" {
		rest.WriteString("\n## Validate\n\n")
		fmt.Fprintf(&rest, "<pre><code>%s</code></pre>\n", escape(ent.Validate))
	}

	if ent.Reviewer.Intro != "" || len(ent.Reviewer.Bullets) > 0 {
		rest.WriteString("\n## Reviewer\n\n")
		if ent.Reviewer.Name != "" {
			fmt.Fprintf(&rest, "`%s`. ", ent.Reviewer.Name)
		}
		if ent.Reviewer.Intro != "" {
			rest.WriteString(ent.Reviewer.Intro + "\n")
		}
		if len(ent.Reviewer.Bullets) > 0 {
			rest.WriteString("\n")
			for _, item := range ent.Reviewer.Bullets {
				fmt.Fprintf(&rest, "- %s\n", item)
			}
		}
	}

	if len(ent.Notes) > 0 {
		rest.WriteString("\n## Notes\n\n")
		for i, n := range ent.Notes {
			if i > 0 {
				rest.WriteString("\n")
			}
			rest.WriteString(n + "\n")
		}
	}

	if len(ent.Also) > 0 {
		rest.WriteString("\n<p class=\"entity-also\">\n")
		for _, link := range ent.Also {
			fmt.Fprintf(&rest, "<a href=\"%s\">%s</a>\n", escape(link.Href), escape(link.Label))
		}
		rest.WriteString("</p>\n")
	}

	return page + rest.String(), nil
}

func mergeBase(root string, schema map[string]any) map[string]any {
	ext := asString(schema["extends"])
	if ext == "" {
		return schema
	}
	basePath := filepath.Join(root, "artifacts/base/schema.json")
	raw, err := os.ReadFile(basePath)
	if err != nil {
		return schema
	}
	var base map[string]any
	if err := json.Unmarshal(raw, &base); err != nil {
		return schema
	}
	baseProps := asMap(get(base, "properties.metadata.properties"))
	childMeta := asMap(schema["metadata"])
	if childMeta == nil {
		return schema
	}
	childProps := asMap(childMeta["properties"])
	if childProps == nil {
		childProps = map[string]any{}
	}
	for k, v := range baseProps {
		if _, ok := childProps[k]; !ok {
			childProps[k] = v
		}
	}
	childMeta["properties"] = childProps
	schema["metadata"] = childMeta
	return schema
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func schemaVersion(schema map[string]any, ent Entity) string {
	if v := asString(get(schema, "metadata.properties.schema_version.const")); v != "" {
		return v
	}
	if v := asString(get(schema, "metadata.properties.schema-version.const")); v != "" {
		return v
	}
	if v := asString(schema["artifact_type"]); v != "" {
		if ver := asString(schema["version"]); strings.HasPrefix(ver, "2.") {
			return v + "/v2"
		}
		return v + "/v1"
	}
	return ent.ID
}

func fieldsTable(schema map[string]any, fields []FieldSpec) string {
	var b strings.Builder
	b.WriteString("<div class=\"entity-table\">\n<table>\n<thead>\n<tr><th>Field</th><th>Required</th><th>Form</th></tr>\n</thead>\n<tbody>\n")
	for _, f := range fields {
		form := f.Form
		if form == "" {
			form = fieldForm(lookupProp(schema, f.Key))
		}
		req := f.Required
		if req == "" {
			req = "no"
		}
		fmt.Fprintf(&b, "<tr><td data-label=\"Field\"><code>%s</code></td><td data-label=\"Required\">%s</td><td data-label=\"Form\">%s</td></tr>\n",
			escape(f.Key), escape(req), formHTML(form))
	}
	b.WriteString("</tbody>\n</table>\n</div>\n")
	return b.String()
}

func sourcesTable(rows []Source) string {
	var b strings.Builder
	b.WriteString("<div class=\"entity-table entity-table-text\">\n<table>\n<thead>\n<tr><th>Work</th><th>Path</th></tr>\n</thead>\n<tbody>\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "<tr><td data-label=\"Work\">%s</td><td data-label=\"Path\">%s</td></tr>\n", escape(row.Work), escape(row.Path))
	}
	b.WriteString("</tbody>\n</table>\n</div>\n")
	return b.String()
}

func statusTable(values []string, kinds, meanings map[string]string) string {
	var b strings.Builder
	b.WriteString("<div class=\"entity-table\">\n<table>\n<thead>\n<tr><th>Value</th><th>Kind</th><th>Meaning</th></tr>\n</thead>\n<tbody>\n")
	for _, v := range values {
		kind := kinds[v]
		if kind == "" {
			kind = "—"
		}
		meaning := meanings[v]
		if meaning == "" {
			meaning = "—"
		}
		fmt.Fprintf(&b, "<tr><td data-label=\"Value\"><code>%s</code></td><td data-label=\"Kind\">%s</td><td data-label=\"Meaning\">%s</td></tr>\n",
			escape(v), escape(kind), escape(meaning))
	}
	b.WriteString("</tbody>\n</table>\n</div>\n")
	return b.String()
}

func illegalBlock(values []string, ent Entity) string {
	items := []string{"Any other status name"}
	set := map[string]bool{}
	for _, v := range values {
		set[v] = true
	}
	if set["replaced"] {
		items = append(items, "`replaced` without `replaced-by`")
	}
	if set["obsoleted"] {
		items = append(items, "`obsoleted` without `obsoleted-by`")
	}
	items = append(items, ent.ExtraIllegal...)
	var b strings.Builder
	b.WriteString("\n<div class=\"entity-illegal\">\n<p>Illegal</p>\n<ul>\n")
	for _, item := range items {
		fmt.Fprintf(&b, "<li>%s</li>\n", mdCode(item))
	}
	b.WriteString("</ul>\n</div>\n")
	return b.String()
}

func sectionTable(schema map[string]any) string {
	required := stringSlice(schema["required_sections"])
	optional := stringSlice(schema["optional_sections"])
	if len(required) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<div class=\"entity-table\">\n<table>\n<thead>\n<tr><th>Section</th><th>Required</th></tr>\n</thead>\n<tbody>\n")
	for _, s := range required {
		fmt.Fprintf(&b, "<tr><td data-label=\"Section\">%s</td><td data-label=\"Required\">yes</td></tr>\n", escape(s))
	}
	for _, s := range optional {
		fmt.Fprintf(&b, "<tr><td data-label=\"Section\">%s</td><td data-label=\"Required\">no</td></tr>\n", escape(s))
	}
	b.WriteString("</tbody>\n</table>\n</div>\n")
	return b.String()
}

func statusValues(schema map[string]any, path string) []string {
	node := get(schema, path)
	if m, ok := node.(map[string]any); ok {
		return stringSlice(m["enum"])
	}
	return nil
}

func lookupProp(schema map[string]any, key string) map[string]any {
	candidates := []string{
		"metadata.properties." + key,
		"nested_blocks." + key,
		"nested_blocks." + strings.ReplaceAll(key, ".", ".properties."),
		"metadata.properties." + strings.ReplaceAll(key, ".", ".properties."),
	}
	for _, c := range candidates {
		if m, ok := get(schema, c).(map[string]any); ok {
			if _, hasEnum := m["enum"]; hasEnum || m["pattern"] != nil || m["const"] != nil || m["type"] != nil || m["description"] != nil {
				return m
			}
			if props, ok := m["properties"].(map[string]any); ok {
				_ = props
			}
		}
	}
	return nil
}

func fieldForm(prop map[string]any) string {
	if prop == nil {
		return "string"
	}
	if v := asString(prop["const"]); v != "" {
		return v
	}
	if enums := stringSlice(prop["enum"]); len(enums) > 0 {
		parts := make([]string, len(enums))
		for i, e := range enums {
			parts[i] = "`" + e + "`"
		}
		return strings.Join(parts, " · ")
	}
	if p := asString(prop["pattern"]); p != "" {
		return humanizePattern(p)
	}
	if t := asString(prop["type"]); t != "" && t != "object" {
		return t
	}
	return "string"
}

func formHTML(form string) string {
	if strings.Contains(form, "`") || strings.Contains(form, " · ") {
		return form
	}
	if form == "See Status" || strings.Contains(form, " ") {
		return escape(form)
	}
	if form == "string" || form == "integer" {
		return escape(form)
	}
	return "<code>" + escape(form) + "</code>"
}

func humanizePattern(p string) string {
	p = strings.TrimPrefix(p, "^")
	p = strings.TrimSuffix(p, "$")
	p = strings.ReplaceAll(p, `\.`, `.`)
	repls := [][2]string{
		{`\d{4}-\d{2}-\d{2}`, `YYYY-MM-DD`},
		{`[0-9]{4}-[0-9]{2}-[0-9]{2}`, `YYYY-MM-DD`},
		{`\d+.\d+.\d+`, `N.N.N`},
		{`[0-9]+.[0-9]+.[0-9]+`, `N.N.N`},
		{`[0-9]{4}`, `NNNN`},
		{`[0-9]{3}`, `NNN`},
		{`[0-9]+`, `NNN`},
		{`\d{4}`, `NNNN`},
		{`[a-z][a-z0-9]*(-[a-z0-9]+)*`, `slug`},
		{`[a-z0-9-]+`, `slug`},
		{`[A-Z0-9-]+`, `ID`},
		{`(BUNDLE-[0-9]+-)?`, `BUNDLE-NNN-`},
		{`(BUNDLE-NNN-)?`, `BUNDLE-NNN-`},
		{`(DIR-[0-9]+-)?`, `DIR-NNN-`},
		{`(DIR-NNN-)?`, `DIR-NNN-`},
		{`(CAP-[0-9]+-)?`, `CAP-NNN-`},
		{`(CAP-NNN-)?`, `CAP-NNN-`},
		{`(\.epic)?`, ``},
		{`(.epic)?`, ``},
	}
	for _, r := range repls {
		p = strings.ReplaceAll(p, r[0], r[1])
	}
	return p
}

func joinFile(dir, name string) string {
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func get(m map[string]any, path string) any {
	if path == "" || m == nil {
		return nil
	}
	cur := any(m)
	for _, part := range strings.Split(path, ".") {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = obj[part]
		if !ok {
			return nil
		}
	}
	return cur
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func stringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func mdCode(s string) string {
	s = escape(s)
	var b strings.Builder
	for {
		start := strings.Index(s, "`")
		if start < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := strings.Index(s[start+1:], "`")
		if end < 0 {
			b.WriteString(s)
			return b.String()
		}
		end = start + 1 + end
		b.WriteString(s[:start])
		b.WriteString("<code>")
		b.WriteString(s[start+1 : end])
		b.WriteString("</code>")
		s = s[end+1:]
	}
}
