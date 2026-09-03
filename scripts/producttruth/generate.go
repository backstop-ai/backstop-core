package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	generatedDir = "docs/_includes/generated"
	regenerate   = "./scripts/generate-product-truth.sh"
)

var stableTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type Manifest struct {
	Version string `yaml:"version"`
	Jobs    []Job  `yaml:"jobs"`
}

type Job struct {
	ID               string   `yaml:"id"`
	Inputs           []string `yaml:"inputs"`
	Output           string   `yaml:"output"`
	OwnerRoute       string   `yaml:"owner_route"`
	OwnerAnchor      string   `yaml:"owner_anchor"`
	Marker           string   `yaml:"marker"`
	Command          string   `yaml:"command"`
	SourceLinkPolicy string   `yaml:"source_link_policy"`
}

type CLIRecord struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Flags       []string `json:"flags"`
}

type SchemaRecord struct {
	ArtifactType    string `json:"artifact_type"`
	PathVersion     string `json:"path_version"`
	DocumentVersion string `json:"document_version"`
	SchemaID        string `json:"schema_id"`
	Title           string `json:"title"`
	Source          string `json:"source"`
}

type PackRecord struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Language    string   `json:"language"`
	Archetype   string   `json:"archetype"`
	Description string   `json:"description"`
	Engines     []string `json:"engines"`
	Repository  string   `json:"repository"`
}

type ReleaseRecord struct {
	Tag          string `json:"tag"`
	Commit       string `json:"commit"`
	CommittedUTC string `json:"committed_utc"`
	Subject      string `json:"subject"`
}

type SourceLinkDescriptor struct {
	Kind          string
	CommitBinding string
	Path          string
	Commit        string
}

func (s SourceLinkDescriptor) MarshalJSON() ([]byte, error) {
	if s.Kind == "commit" {
		return json.Marshal(struct {
			Kind          string `json:"kind"`
			CommitBinding string `json:"commit_binding"`
			Commit        string `json:"commit"`
		}{s.Kind, s.CommitBinding, s.Commit})
	}
	return json.Marshal(struct {
		Kind          string `json:"kind"`
		CommitBinding string `json:"commit_binding"`
		Path          string `json:"path"`
	}{s.Kind, s.CommitBinding, s.Path})
}

type RenderedJob struct {
	Job   Job
	Bytes []byte
}

type Drift struct {
	Job    Job
	Detail string
}

func (d Drift) Error() string {
	return diagnostic("PT202_DRIFT", d.Job.ID, d.Job.Output, d.Job.Inputs, d.Detail).Error()
}

type productTruthError struct{ message string }

func (e productTruthError) Error() string { return e.message }

func diagnostic(code, job, output string, inputs []string, detail string) error {
	inputText := "-"
	if len(inputs) != 0 {
		inputText = strings.Join(inputs, ",")
	}
	return productTruthError{fmt.Sprintf("product-truth[%s] job=%s output=%s inputs=%s: %s", code, job, output, inputText, detail)}
}

func LoadManifest(root, path string) (Manifest, error) {
	var manifest Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, diagnostic("PT001_MANIFEST", "pipeline", path, nil, err.Error())
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, diagnostic("PT001_MANIFEST", "pipeline", path, nil, err.Error())
	}
	if manifest.Version != "product-truth/v1" || len(manifest.Jobs) != 4 {
		return manifest, diagnostic("PT001_MANIFEST", "pipeline", path, nil, "manifest must contain product-truth/v1 and exactly four jobs")
	}
	expected := []string{"cli-command-catalog", "artifact-schema-catalog", "published-pack-catalog", "release-history"}
	seenID, seenOutput, seenOwner := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i, job := range manifest.Jobs {
		owner := job.OwnerRoute + "#" + job.OwnerAnchor
		if job.ID != expected[i] || seenID[job.ID] || seenOutput[job.Output] || seenOwner[owner] {
			return manifest, diagnostic("PT001_MANIFEST", job.ID, job.Output, job.Inputs, "job order, ID, output, and owner must match the closed four-job inventory")
		}
		if job.Marker != "GENERATED PRODUCT TRUTH" || job.Command != regenerate || len(job.Inputs) == 0 || job.SourceLinkPolicy == "" {
			return manifest, diagnostic("PT001_MANIFEST", job.ID, job.Output, job.Inputs, "incomplete job declaration")
		}
		if err := validateContainedPath(root, job.Output); err != nil {
			return manifest, diagnostic("PT001_MANIFEST", job.ID, job.Output, job.Inputs, err.Error())
		}
		seenID[job.ID], seenOutput[job.Output], seenOwner[owner] = true, true, true
	}
	return manifest, nil
}

func RenderAll(root string, manifest Manifest) ([]RenderedJob, error) {
	rendered := make([]RenderedJob, 0, len(manifest.Jobs))
	for _, job := range manifest.Jobs {
		var records any
		var links []SourceLinkDescriptor
		var headers []string
		var rows [][]string
		var err error
		switch job.ID {
		case "cli-command-catalog":
			var typed []CLIRecord
			typed, err = loadCLI(root)
			records, links = typed, []SourceLinkDescriptor{{Kind: "tree", CommitBinding: "site", Path: "cmd/backstop"}}
			headers, rows = cliRows(typed)
		case "artifact-schema-catalog":
			var typed []SchemaRecord
			typed, err = loadSchemas(root)
			records, links = typed, schemaLinks(typed)
			headers, rows = schemaRows(typed)
		case "published-pack-catalog":
			var typed []PackRecord
			typed, err = loadPublishedPacks(root)
			records = typed
			links = []SourceLinkDescriptor{{Kind: "blob", CommitBinding: "site", Path: "docs/_data/published-pack-inventory.yml"}}
			headers, rows = packRows(typed)
		case "release-history":
			var typed []ReleaseRecord
			typed, err = loadReleases(root)
			records, links = typed, releaseLinks(typed)
			headers, rows = releaseRows(typed)
		default:
			err = errors.New("unknown job")
		}
		if err != nil {
			return nil, err
		}
		contents, err := renderJob(job, records, links, headers, rows)
		if err != nil {
			return nil, diagnostic("PT001_MANIFEST", job.ID, job.Output, job.Inputs, err.Error())
		}
		rendered = append(rendered, RenderedJob{Job: job, Bytes: contents})
	}
	return rendered, nil
}

func loadCLI(root string) ([]CLIRecord, error) {
	tool, err := exec.LookPath("go")
	if err != nil {
		return nil, diagnostic("PT101_COMMAND", "cli-command-catalog", "docs/_includes/generated/cli-command-catalog.md", []string{"cmd/backstop"}, "go tool not found")
	}
	cmd := exec.Command(tool, "run", "./cmd/backstop", "commands")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "TZ=UTC")
	out, err := cmd.Output()
	if err != nil {
		return nil, diagnostic("PT101_COMMAND", "cli-command-catalog", "docs/_includes/generated/cli-command-catalog.md", []string{"cmd/backstop"}, err.Error())
	}
	var records []CLIRecord
	if err := json.Unmarshal(out, &records); err != nil {
		return nil, diagnostic("PT101_COMMAND", "cli-command-catalog", "docs/_includes/generated/cli-command-catalog.md", []string{"cmd/backstop"}, err.Error())
	}
	for i := range records {
		sort.Strings(records[i].Flags)
		if !validScalar(records[i].Name) || !validScalar(records[i].Path) || !validScalar(records[i].Description) {
			return nil, diagnostic("PT101_COMMAND", "cli-command-catalog", "docs/_includes/generated/cli-command-catalog.md", []string{"cmd/backstop"}, "invalid command scalar")
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Path != records[j].Path {
			return records[i].Path < records[j].Path
		}
		return strings.Join(records[i].Flags, "\x00") < strings.Join(records[j].Flags, "\x00")
	})
	return records, nil
}

func loadSchemas(root string) ([]SchemaRecord, error) {
	paths, err := filepath.Glob(filepath.Join(root, "artifacts", "*", "v*", "schema.json"))
	if err != nil {
		return nil, err
	}
	paths = append(paths, filepath.Join(root, "artifacts", "base", "schema.json"))
	sort.Strings(paths)
	records := make([]SchemaRecord, 0, len(paths))
	for _, absolute := range paths {
		rel, relErr := filepath.Rel(root, absolute)
		if relErr != nil {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{absolute}, relErr.Error())
		}
		data, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{filepath.ToSlash(rel)}, readErr.Error())
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{filepath.ToSlash(rel)}, err.Error())
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		artifactType, pathVersion := "base", "base"
		if len(parts) == 4 {
			artifactType, pathVersion = parts[1], parts[2]
		}
		if declared, ok := raw["artifact_type"]; ok && declared != artifactType {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{filepath.ToSlash(rel)}, "artifact_type disagrees with path")
		}
		documentVersion := "not-declared"
		if value, ok := scalarString(raw["version"]); ok {
			documentVersion = value
		} else if artifactType != "base" && artifactType != "backstop-yml" {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{filepath.ToSlash(rel)}, "version must be a scalar")
		}
		schemaID, okID := scalarString(raw["$id"])
		title, okTitle := scalarString(raw["title"])
		if !okID || !okTitle {
			return nil, diagnostic("PT102_SCHEMA", "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", []string{filepath.ToSlash(rel)}, "$id and title must be scalar strings")
		}
		records = append(records, SchemaRecord{artifactType, pathVersion, documentVersion, schemaID, title, filepath.ToSlash(rel)})
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].ArtifactType == "base" {
			return records[j].ArtifactType != "base"
		}
		if records[j].ArtifactType == "base" {
			return false
		}
		if records[i].ArtifactType != records[j].ArtifactType {
			return records[i].ArtifactType < records[j].ArtifactType
		}
		return versionNumber(records[i].PathVersion) < versionNumber(records[j].PathVersion)
	})
	return records, nil
}

func loadPublishedPacks(root string) ([]PackRecord, error) {
	const (
		jobID     = "published-pack-catalog"
		output    = "docs/_includes/generated/published-pack-catalog.md"
		inventory = "docs/_data/published-pack-inventory.yml"
	)
	inputs := []string{inventory}
	data, err := os.ReadFile(filepath.Join(root, inventory))
	if err != nil {
		return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, err.Error())
	}
	var parsed struct {
		Version      string `yaml:"version"`
		Organization string `yaml:"organization"`
		Packs        []struct {
			Name        string   `yaml:"name"`
			Version     string   `yaml:"version"`
			Language    string   `yaml:"language"`
			Archetype   string   `yaml:"archetype"`
			Description string   `yaml:"description"`
			Engines     []string `yaml:"engines"`
			Repository  string   `yaml:"repository"`
		} `yaml:"packs"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&parsed); err != nil {
		return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, err.Error())
	}
	if parsed.Version != "published-pack-inventory/v1" || parsed.Organization != "backstop-ai" || len(parsed.Packs) == 0 {
		return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, "inventory must declare published-pack-inventory/v1 for backstop-ai with at least one pack")
	}
	namePattern := regexp.MustCompile(`^backstop-ai/[a-z0-9-]+$`)
	versionPattern := regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	seen := map[string]bool{}
	records := make([]PackRecord, 0, len(parsed.Packs))
	for _, pack := range parsed.Packs {
		engines := append([]string(nil), pack.Engines...)
		sort.Strings(engines)
		record := PackRecord{pack.Name, pack.Version, pack.Language, pack.Archetype, pack.Description, engines, pack.Repository}
		if seen[record.Name] || !namePattern.MatchString(record.Name) || !versionPattern.MatchString(record.Version) {
			return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, "invalid or duplicate pack identity for "+pack.Name)
		}
		if !validScalar(record.Language) || !validScalar(record.Archetype) || !validScalar(record.Description) || !validScalar(record.Repository) {
			return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, "invalid pack scalar for "+pack.Name)
		}
		if record.Repository != "https://github.com/"+record.Name {
			return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, "repository must be the GitHub URL for "+pack.Name)
		}
		for _, engine := range engines {
			if !validScalar(engine) {
				return nil, diagnostic("PT103_PACK_INVENTORY", jobID, output, inputs, "invalid engine name for "+pack.Name)
			}
		}
		seen[record.Name] = true
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return records, nil
}

func loadReleases(root string) ([]ReleaseRecord, error) {
	out, err := gitOutput(root, "for-each-ref", "--format=%(refname:strip=2)", "refs/tags")
	if err != nil {
		return nil, diagnostic("PT104_GIT_REF", "release-history", "docs/_includes/generated/release-history.md", []string{"refs/tags/vMAJOR.MINOR.PATCH"}, err.Error())
	}
	var records []ReleaseRecord
	for _, tag := range strings.Split(stringTrimSpace(string(out)), "\n") {
		if tag == "" || !stableTagPattern.MatchString(tag) {
			continue
		}
		commit, peelErr := gitOutput(root, "rev-parse", "--verify", "--end-of-options", "refs/tags/"+tag+"^{commit}")
		if peelErr != nil {
			return nil, diagnostic("PT104_GIT_REF", "release-history", "docs/_includes/generated/release-history.md", []string{tag}, peelErr.Error())
		}
		sha := stringTrimSpace(string(commit))
		if err := exec.Command("git", "-C", root, "merge-base", "--is-ancestor", sha, "HEAD").Run(); err != nil {
			continue
		}
		meta, showErr := gitOutput(root, "show", "-s", "--format=%cI%x00%s", sha)
		if showErr != nil {
			return nil, diagnostic("PT104_GIT_REF", "release-history", "docs/_includes/generated/release-history.md", []string{tag}, showErr.Error())
		}
		parts := strings.SplitN(stringTrimSpace(string(meta)), "\x00", 2)
		if len(parts) != 2 {
			return nil, diagnostic("PT104_GIT_REF", "release-history", "docs/_includes/generated/release-history.md", []string{tag}, "invalid git show response")
		}
		parsed, dateErr := time.Parse(time.RFC3339, parts[0])
		if dateErr != nil {
			return nil, dateErr
		}
		records = append(records, ReleaseRecord{tag, sha, parsed.UTC().Format(time.RFC3339), parts[1]})
	}
	sort.Slice(records, func(i, j int) bool { return semverGreater(records[i].Tag, records[j].Tag) })
	return records, nil
}

func renderJob(job Job, records any, links []SourceLinkDescriptor, headers []string, rows [][]string) ([]byte, error) {
	owner := job.OwnerRoute + "#" + job.OwnerAnchor
	envelope := struct {
		Job         string                 `json:"job"`
		Output      string                 `json:"output"`
		Owner       string                 `json:"owner"`
		Records     any                    `json:"records"`
		SourceLinks []SourceLinkDescriptor `json:"source_links"`
	}{job.ID, job.Output, owner, records, links}
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(envelope); err != nil {
		return nil, err
	}
	digestBytes := sha256.Sum256(canonical.Bytes())
	digest := hex.EncodeToString(digestBytes[:])
	var out strings.Builder
	fmt.Fprintf(&out, "<!-- GENERATED PRODUCT TRUTH | job=%s | inputs=%s | owner=%s | regenerate=%s | DO NOT EDIT -->\n", job.ID, strings.Join(job.Inputs, ","), owner, regenerate)
	fmt.Fprintf(&out, "<!-- PRODUCT-TRUTH:BEGIN job=%s digest=sha256:%s -->\n", job.ID, digest)
	fmt.Fprintf(&out, "<table data-product-truth-job=\"%s\">\n<thead><tr>", job.ID)
	for _, header := range headers {
		fmt.Fprintf(&out, "<th>%s</th>", escapeCell(header))
	}
	out.WriteString("</tr></thead>\n<tbody>\n")
	for _, row := range rows {
		out.WriteString("<tr>")
		for _, cell := range row {
			fmt.Fprintf(&out, "<td>%s</td>", escapeCell(cell))
		}
		out.WriteString("</tr>\n")
	}
	out.WriteString("</tbody>\n</table>\n")
	fmt.Fprintf(&out, "<!-- PRODUCT-TRUTH:SOURCES-BEGIN job=%s owner=%s digest=sha256:%s -->\n", job.ID, owner, digest)
	fmt.Fprintf(&out, "<ul data-generated-source-descriptors data-product-truth-job=\"%s\">\n", job.ID)
	for _, link := range links {
		if link.Kind == "commit" {
			fmt.Fprintf(&out, "<li data-generated-source-descriptor data-source-kind=\"commit\" data-commit-binding=\"record\" data-source-commit=\"%s\">https://github.com/backstop-ai/backstop-core/commit/%s</li>\n", escapeCell(link.Commit), escapeCell(link.Commit))
		} else {
			fmt.Fprintf(&out, "<li data-generated-source-descriptor data-source-kind=\"%s\" data-commit-binding=\"site\" data-source-path=\"%s\">https://github.com/backstop-ai/backstop-core/%s/&lt;SITE-COMMIT&gt;/%s</li>\n", link.Kind, escapeCell(link.Path), map[string]string{"tree": "tree", "blob": "blob"}[link.Kind], escapeCell(link.Path))
		}
	}
	out.WriteString("</ul>\n")
	fmt.Fprintf(&out, "<!-- PRODUCT-TRUTH:SOURCES-END job=%s -->\n", job.ID)
	fmt.Fprintf(&out, "<!-- PRODUCT-TRUTH:END job=%s -->\n", job.ID)
	return []byte(out.String()), nil
}

func CheckAll(root string, rendered []RenderedJob) ([]Drift, error) {
	if transactionExists(root) {
		return nil, diagnostic("PT203_TRANSACTION", "pipeline", "-", nil, "transaction recovery required")
	}
	var drifts []Drift
	registered := map[string]bool{}
	for _, item := range rendered {
		registered[filepath.Base(item.Job.Output)] = true
		actual, err := os.ReadFile(filepath.Join(root, item.Job.Output))
		if err != nil {
			drifts = append(drifts, Drift{item.Job, "missing file; regenerate with " + regenerate})
			continue
		}
		if !bytes.Equal(actual, item.Bytes) {
			drifts = append(drifts, Drift{item.Job, firstDifference(actual, item.Bytes) + "; regenerate with " + regenerate})
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, generatedDir))
	if err != nil && !os.IsNotExist(err) {
		return drifts, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && !registered[entry.Name()] {
			drifts = append(drifts, Drift{Job: Job{ID: "pipeline", Output: filepath.ToSlash(filepath.Join(generatedDir, entry.Name()))}, Detail: "unregistered generated file"})
		}
	}
	if len(drifts) > 0 {
		return drifts, errors.New("product truth drift detected")
	}
	return nil, nil
}

func VerifySourceIncludes(root string, manifest Manifest) error {
	owners := map[string]struct{ File, Title string }{
		"cli-command-catalog":     {"docs/reference.md", "CLI command catalog"},
		"artifact-schema-catalog": {"docs/reference.md", "Artifact schema catalog"},
		"published-pack-catalog":  {"docs/pack/examples.md", "Published pack catalog"},
		"release-history":         {"docs/status.md", "Release history"},
	}
	for _, job := range manifest.Jobs {
		owner := owners[job.ID]
		data, err := os.ReadFile(filepath.Join(root, owner.File))
		if err != nil {
			return diagnostic("PT204_CONSUMPTION", job.ID, job.Output, job.Inputs, err.Error())
		}
		heading := "## " + owner.Title + " {#" + job.OwnerAnchor + "}"
		include := "<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=" + job.ID + " -->\n{% include generated/" + filepath.Base(job.Output) + " %}\n<!-- PRODUCT-TRUTH-INCLUDE:END job=" + job.ID + " -->"
		bare := heading + "\n\n" + include
		wrapped := heading + "\n\n<section data-generated-region data-product-truth-job=\"" + job.ID + "\">\n" + include + "\n</section>"
		source := string(data)
		if strings.Count(source, bare)+strings.Count(source, wrapped) != 1 || strings.Count(source, include) != 1 {
			return diagnostic("PT204_CONSUMPTION", job.ID, job.Output, job.Inputs, "owner include must occur exactly once in "+owner.File)
		}
	}
	return nil
}

func validateContainedPath(root, rel string) error {
	if filepath.IsAbs(rel) || rel == "" {
		return errors.New("path must be repository-relative")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path escapes repository")
	}
	abs := filepath.Join(root, clean)
	relative, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(relative, "..") {
		return errors.New("path escapes repository")
	}
	return nil
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;", "`", "&#96;", "|", "&#124;", "\n", "<br>")
	return r.Replace(value)
}
func validScalar(value string) bool {
	if value == "" || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	for _, r := range value {
		if r < 0x20 && r != '\t' {
			return false
		}
	}
	return true
}
func scalarString(value any) (string, bool) { s, ok := value.(string); return s, ok && validScalar(s) }
func versionNumber(value string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(value, "v"))
	if err != nil {
		return -1
	}
	return n
}
func stringTrimSpace(value string) string { return strings.Trim(value, " \t\r\n") }
func gitOutput(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	return cmd.Output()
}
func semverGreater(left, right string) bool {
	l := stableTagPattern.FindStringSubmatch(left)
	r := stableTagPattern.FindStringSubmatch(right)
	for i := 1; i <= 3; i++ {
		li, leftErr := strconv.Atoi(l[i])
		ri, rightErr := strconv.Atoi(r[i])
		if leftErr != nil || rightErr != nil {
			return false
		}
		if li != ri {
			return li > ri
		}
	}
	return false
}
func firstDifference(actual, expected []byte) string {
	a, e := strings.Split(string(actual), "\n"), strings.Split(string(expected), "\n")
	limit := len(a)
	if len(e) < limit {
		limit = len(e)
	}
	for i := 0; i < limit; i++ {
		if a[i] != e[i] {
			return fmt.Sprintf("first differing line %d", i+1)
		}
	}
	return fmt.Sprintf("first differing line %d", limit+1)
}
func cliRows(records []CLIRecord) ([]string, [][]string) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		flags := "—"
		if len(r.Flags) > 0 {
			flags = strings.Join(r.Flags, "\n")
		}
		rows = append(rows, []string{r.Name, r.Path, r.Description, flags})
	}
	return []string{"Command", "Path", "Description", "Flags"}, rows
}
func schemaRows(records []SchemaRecord) ([]string, [][]string) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, []string{r.ArtifactType, r.PathVersion, r.DocumentVersion, r.SchemaID, r.Title, r.Source})
	}
	return []string{"Artifact type", "Schema path version", "Document version", "Schema ID", "Title", "Source"}, rows
}
func packRows(records []PackRecord) ([]string, [][]string) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		engines := strings.Join(r.Engines, ", ")
		if engines == "" {
			engines = "—"
		}
		covers := r.Language + " " + r.Archetype + ": " + r.Description
		rows = append(rows, []string{r.Name, r.Version, covers, engines, r.Repository})
	}
	return []string{"Pack", "Version", "Covers", "Engines", "Source"}, rows
}
func releaseRows(records []ReleaseRecord) ([]string, [][]string) {
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, []string{r.Tag, r.Commit, r.CommittedUTC, r.Subject})
	}
	return []string{"Version", "Commit", "Committed UTC", "Subject"}, rows
}
func schemaLinks(records []SchemaRecord) []SourceLinkDescriptor {
	links := make([]SourceLinkDescriptor, 0, len(records))
	for _, r := range records {
		links = append(links, SourceLinkDescriptor{Kind: "blob", CommitBinding: "site", Path: r.Source})
	}
	return links
}
func releaseLinks(records []ReleaseRecord) []SourceLinkDescriptor {
	links := make([]SourceLinkDescriptor, 0, len(records))
	for _, r := range records {
		links = append(links, SourceLinkDescriptor{Kind: "commit", CommitBinding: "record", Commit: r.Commit})
	}
	return links
}
