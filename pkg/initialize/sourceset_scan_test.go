package initialize

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sourceset_scan_test.go holds the DENYLIST claims, whose entire content is that a
// literal is ABSENT.
//
// ★ THEY ARE WRITTEN LAST ON PURPOSE. A scan authored before the source exists is a
// vacuous green that only bites later — it passes over an empty set and keeps passing
// while the thing it forbids is added somewhere it does not look.
//
// ★ SHARP EDGE 10 — THE SCAN BOUNDARY IS THE CLAIM. Every scan below enumerates the
// init source set from a GLOB at test time (`pkg/initialize/**` plus
// `cmd/backstop/init*.go`, excluding `_test.go`) and asserts the enumeration is
// NON-EMPTY before scanning. A future implementer who moves init logic into a helper
// OUTSIDE that boundary would otherwise silently EMPTY these claims without failing
// one. initSourceSet (preserve_test.go) is that enumeration, and it fails loudly on an
// empty result.
//
// ★ CODE VERSUS PROSE. A denylist over raw file text cannot tell a forbidden CALL from
// a comment EXPLAINING why the call is forbidden — and these files deliberately carry
// a lot of the latter. So the structural scans read COMMENT-STRIPPED code, and only the
// claims genuinely about the whole text (a language noun anywhere) read the raw bytes.

// initSourceCode returns each init source file's code with COMMENTS REMOVED, so a scan
// for a forbidden construct cannot fire on a comment that names it in order to forbid
// it.
func initSourceCode(t *testing.T) map[string]string {
	t.Helper()

	stripped := map[string]string{}
	for _, path := range initSourceSet(t) {
		fset := token.NewFileSet()
		// Parsed WITHOUT ParseComments, so the printed form carries none.
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		var out strings.Builder
		if printErr := printer.Fprint(&out, fset, parsed); printErr != nil {
			t.Fatalf("printing %s: %v", path, printErr)
		}
		stripped[path] = out.String()
	}
	if len(stripped) == 0 {
		t.Fatal("the comment-stripped source set is empty; every scan below would pass trivially")
	}
	return stripped
}

// initSourceText returns each init source file's RAW text, comments included.
func initSourceText(t *testing.T) map[string]string {
	t.Helper()
	raw := map[string]string{}
	for _, path := range initSourceSet(t) {
		raw[path] = readWholeFile(t, path)
	}
	return raw
}

// readWholeFile reads a file or fails the test.
func readWholeFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

// assertAbsentFromCode fails when any comment-stripped source file matches pattern.
func assertAbsentFromCode(t *testing.T, pattern, why string) {
	t.Helper()
	matcher := regexp.MustCompile(pattern)
	for path, code := range initSourceCode(t) {
		if match := matcher.FindString(code); match != "" {
			t.Fatalf("%s carries %q in its CODE.\n%s", filepath.Base(path), match, why)
		}
	}
}

// assertAbsentFromText fails when any raw source file matches pattern.
func assertAbsentFromText(t *testing.T, pattern, why string) {
	t.Helper()
	matcher := regexp.MustCompile(pattern)
	for path, body := range initSourceText(t) {
		if match := matcher.FindString(body); match != "" {
			t.Fatalf("%s carries %q.\n%s", filepath.Base(path), match, why)
		}
	}
}

// TestInit_SourceSetHoldsNoLanguageFrameworkOrPlatformLiteral (SPEC-069 CLM-044).
//
// THE TEETH OF THE ZERO-DETECTION INVARIANT. No language name, framework name,
// ecosystem-manifest filename, or CI-platform name may appear in the init source set —
// in code OR in prose. That knowledge lives in PACKS as data and in the consumer's own
// arguments, and reaches init only through pack manifests, recipe refs and flags.
//
// The forbidden nouns are enumerated in THIS TEST, not in the source, which is what
// makes naming them here an assertion rather than the very bake they forbid.
func TestInit_SourceSetHoldsNoLanguageFrameworkOrPlatformLiteral(t *testing.T) {
	forbidden := map[string]string{
		"language names":           `(?i)\b(golang|typescript|javascript|python|rust|ruby|kotlin|swift|dotnet|csharp|php|scala|elixir)\b`,
		"package managers":         `(?i)\b(npm|pnpm|yarn|cargo|maven|gradle|pipenv|poetry|bundler|composer|nuget)\b`,
		"frameworks":               `(?i)\b(react|vue|angular|django|rails|spring boot|express\.js|nextjs|svelte)\b`,
		"ecosystem manifest files": `(?i)(package\.json|go\.mod|cargo\.toml|requirements\.txt|pyproject\.toml|gemfile|pom\.xml|build\.gradle|composer\.json)`,
		"ecosystem directories":    `(?i)(node_modules|__pycache__|\.venv|site-packages)`,
		"CI platform names":        `(?i)(github actions|gitlab[- ]?ci|jenkinsfile|circleci|bitbucket[- ]?pipelines|travis[- ]?ci|azure pipelines)`,
	}

	for kind, pattern := range forbidden {
		t.Run(kind, func(t *testing.T) {
			assertAbsentFromText(t, pattern,
				"Init performs ZERO language, framework, ecosystem or CI-platform detection. That knowledge belongs in a pack as DATA or in the consumer's own arguments — a literal here is the bake this invariant exists to prevent.")
		})
	}
}

// TestInit_AuthorsNoSourceFileContentOfItsOwn (SPEC-069 CLM-048).
//
// ★ THE WORDING MATTERS AND AN EARLIER SPEC DRAFT GOT IT WRONG (Sharp Edge 21). The
// denylist is on core AUTHORING the content, NEVER on init DELIVERING it through a pack
// recipe. A test asserting "init writes no source file" would re-delete the scaffold
// step and PASS while doing it — which is exactly how a requirement gets deleted by a
// rewording with a green suite watching.
//
// So this scans for a source-file PAYLOAD, template body, filename or extension
// literal, and separately asserts the scaffold step is STILL PRESENT and still reaches
// the applier.
func TestInit_AuthorsNoSourceFileContentOfItsOwn(t *testing.T) {
	t.Run("no source-file payload or template body", func(t *testing.T) {
		// A source-file extension in a string literal would be core naming a file it
		// intends to write. The recipe names its own target; init contributes no path.
		assertAbsentFromCode(t, `"[^"]*\.(go|ts|tsx|js|jsx|py|rs|rb|java|kt|swift|c|cpp|h)"`,
			"Core must AUTHOR no source-file content: no payload, no template body, no filename and no extension. The scaffolded file comes from a pack recipe.")
		assertAbsentFromCode(t, `(?i)(text/template|html/template)`,
			"Init contains no template engine. Rendering is the recipe mechanism's, and init must not interpret, rewrite or language-specialize template content.")
	})

	t.Run("the scaffold step still DELIVERS one", func(t *testing.T) {
		// The falsifying half. Without it, deleting the scaffold step outright would
		// make every assertion above pass.
		code := initSourceCode(t)
		scaffold, present := code[findSourcePath(t, code, "step_scaffold.go")]
		if !present {
			t.Fatal("there is no scaffold step at all; REQ-009 requires init to DELIVER a first source file through a pack recipe, and an absent step passes every denylist above")
		}
		if !strings.Contains(scaffold, "applier.Apply") {
			t.Fatal("the scaffold step does not reach the recipe applier; what REQ-009 forbids is core AUTHORING the file, and what it REQUIRES is init DELIVERING one through a pack")
		}
	})
}

// findSourcePath returns the enumerated path whose base name matches, or fails.
func findSourcePath(t *testing.T, code map[string]string, base string) string {
	t.Helper()
	for path := range code {
		if filepath.Base(path) == base {
			return path
		}
	}
	t.Fatalf("the init source set contains no %s", base)
	return ""
}

// TestInit_SourceSetHoldsNoPackNameLiteralOrDefaultRoster (SPEC-069 CLM-051).
//
// Sharp Edge 7: "the omakase base bundle" is the seven-capability STEP set, NOT a baked
// pack roster. A roster in core would be the same defect one layer up from core
// supplying half a recipe ref — and packs enter a project only through an explicit
// consumer act.
func TestInit_SourceSetHoldsNoPackNameLiteralOrDefaultRoster(t *testing.T) {
	// An `<org>/<name>` string literal is the shape a pack name takes. IMPORT PATHS are
	// quoted too and are the same shape, so they are excluded STRUCTURALLY — by walking
	// the AST and skipping import specs — rather than by a prefix guess that would also
	// hide a real pack name someone prefixed to look like one.
	matcher := regexp.MustCompile(`^"[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*(@.*)?"$`)
	for _, path := range initSourceSet(t) {
		for _, literal := range nonImportStringLiterals(t, path) {
			if matcher.MatchString(literal) {
				t.Fatalf("%s carries the pack-name-shaped literal %s. Core holds NO pack name and no default roster: there is nothing for init to install that a consumer did not name.",
					filepath.Base(path), literal)
			}
		}
	}

	// And no roster-shaped collection of them.
	assertAbsentFromCode(t, `(?i)(defaultPacks|packRoster|basePacks|bundledPacks|recommendedPacks)`,
		"A default pack roster in core is the same defect as core constructing the pack half of a recipe ref, one layer up.")
}

// TestInit_SourceSetConstructsNoPartOfTheCIRef (SPEC-069 CLM-073).
//
// No pack name, no recipe id, no version literal, no CI-platform name. The whole `--ci`
// value is OPAQUE to core and is handed to the shipped resolve+apply path VERBATIM,
// which is precisely what makes a consumer naming an entirely different pack equally
// valid.
func TestInit_SourceSetConstructsNoPartOfTheCIRef(t *testing.T) {
	assertAbsentFromCode(t, `"[0-9]+\.[0-9]+\.[0-9]+"`,
		"A version literal in core is half a recipe ref. The pin is the consumer's, always.")
	assertAbsentFromCode(t, `(?i)"[^"]*:[^"]*@[0-9]`,
		"A fully-formed recipe ref in core is core constructing what only the consumer may supply.")

	// The ref reaches the applier WHOLE. A reassembled ref would mean core had taken it
	// apart, which is the same defect as constructing one from nothing.
	code := initSourceCode(t)
	ci := code[findSourcePath(t, code, "step_ci.go")]
	if !strings.Contains(ci, "applier.Apply(projectRoot, ref)") {
		t.Fatal("the CI step does not hand the applier the ORIGINAL ref value; a ref rebuilt from parsed parts is core constructing a ref")
	}
}

// TestInit_SourceSetConstructsNoPartOfTheScaffoldRefOrItsPayload (SPEC-069 CLM-131).
//
// The same denylist as the CI ref, PLUS no source-file payload, template body, filename
// or extension. `--scaffold` follows `--ci`'s governance shape exactly, so it inherits
// the same prohibition and adds the one that is specific to producing a file.
func TestInit_SourceSetConstructsNoPartOfTheScaffoldRefOrItsPayload(t *testing.T) {
	assertAbsentFromCode(t, `(?i)(defaultScaffold|scaffoldRoster|builtinScaffold|scaffoldTemplate)`,
		"Core holds no default scaffold recipe and no scaffold-pack roster.")

	code := initSourceCode(t)
	scaffold := code[findSourcePath(t, code, "step_scaffold.go")]
	if !strings.Contains(scaffold, "applier.Apply(projectRoot, ref)") {
		t.Fatal("the scaffold step does not hand the applier the ORIGINAL ref value")
	}
	// NO PAYLOAD BYTES OF ITS OWN. The discriminator is not LENGTH — this step's report
	// wording is legitimately long, and a length threshold would be a heuristic that
	// fires on prose and misses a short payload. It is SHAPE: a file payload is
	// MULTI-LINE, and every string this step holds is a single-line report sentence.
	for _, literal := range nonImportStringLiterals(t, findSourcePath(t, code, "step_scaffold.go")) {
		if strings.Contains(literal, `\n`) {
			t.Fatalf("the scaffold step carries the multi-line literal %s, which is the shape a baked file payload takes. The scaffolded file's bytes come from the pack's recipe payload, never from here.", literal)
		}
	}
}

// TestInit_SuppliesNoRecipeParamToEitherApply (SPEC-069 CLM-136).
//
// No param map is constructed and no value is derived from the target directory's name
// or from anything init found on disk (Sharp Edge 20). The tempting fix — passing the
// project basename already computed for `project:` — is core constructing recipe INPUT,
// the same defect one layer in as core constructing half a ref.
func TestInit_SuppliesNoRecipeParamToEitherApply(t *testing.T) {
	// The seam itself has NO param surface, which is what makes this true by
	// construction rather than by discipline.
	code := initSourceCode(t)
	seams := code[findSourcePath(t, code, "seams.go")]
	if !strings.Contains(seams, "Apply(projectRoot string, ref string) (ApplyOutcome, error)") {
		t.Fatal("the RecipeApplier seam's signature has changed; a param argument on it is where a derived value would enter")
	}
	assertAbsentFromCode(t, `(?i)(suppliedParams|recipeParams|paramValues|--param)`,
		"Init supplies no recipe param to either apply. A recipe declaring a param required with no default cannot be applied by init at all, and the shipped apply's own error surfaces verbatim.")
	assertAbsentFromCode(t, `map\[string\]string\{[^}]*\}`,
		"A string map literal in the init source set is the shape a params map takes.")
}

// TestInit_ImplementsNoBaselineSeedingMachinery (SPEC-069 CLM-060).
//
// Nothing writes `.backstop/baseline.json` or computes a fingerprint. That machinery is
// ISSUE-056's, and this spec builds none of it — the baseline step is a delegation seam
// and the production seeder's whole body returns a sentinel.
//
// The scan is over WRITES and HASHING rather than over the literal path: the path
// legitimately appears as a `.gitignore` ENTRY and in the prose that explains the
// boundary, and a scan that fired on those would be unfixable without deleting the
// explanation.
func TestInit_ImplementsNoBaselineSeedingMachinery(t *testing.T) {
	t.Run("no hashing", func(t *testing.T) {
		assertAbsentFromCode(t, `(?i)(crypto/sha|crypto/md5|hash\.|sha256|sha1\.|Sum256|fnv\.)`,
			"Computing a fingerprint is the other half of baseline seeding, and this command builds neither half.")
	})

	t.Run("no write reaches a baseline path", func(t *testing.T) {
		for path, code := range initSourceCode(t) {
			for _, line := range strings.Split(code, "\n") {
				if !strings.Contains(line, "baseline") {
					continue
				}
				for _, write := range []string{"os.WriteFile", "os.Create", "os.OpenFile", "io.Copy"} {
					if strings.Contains(line, write) {
						t.Fatalf("%s writes to a baseline path: %s\nThe gitignored local baseline is owned by ISSUE-056; this command designs and builds none of that machinery.",
							filepath.Base(path), strings.TrimSpace(line))
					}
				}
			}
		}
	})
}

// TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage (SPEC-069
// CLM-063).
//
// Two halves. This implementation changes NO file under pkg/gate — asserted against
// GIT, because that is the only place "was this file changed" is actually recorded —
// and init neither rewrites, suppresses nor substitutes for the remoteless
// `baseline_comparison` message.
func TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage(t *testing.T) {
	t.Run("no file under pkg/gate is modified", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git is not on PATH; this half of the claim is recorded in git and nowhere else")
		}
		repo := repositoryRoot(t)

		// ★ THE BASELINE IS THE WORKING TREE VERSUS HEAD, NOT A MERGE BASE, AND THE
		// DIFFERENCE IS ATTRIBUTION. A merge-base diff carries every commit anyone
		// landed since the branch point — in a shared working tree that includes other
		// sessions' work under pkg/gate, which this claim would then blame on THIS
		// implementation. The working tree against HEAD is exactly this change set.
		status := exec.Command("git", "status", "--porcelain", "--", "pkg/gate")
		status.Dir = repo
		changed, err := status.Output()
		if err != nil {
			t.Skipf("git could not report the working tree: %v", err)
		}
		if strings.TrimSpace(string(changed)) != "" {
			t.Fatalf("this implementation changed files under pkg/gate:\n%s\nREQ-013 forbids it: the self-consistency of the remoteless baseline_comparison message is pure gate machinery owned elsewhere.",
				changed)
		}

		// NON-VACUITY. Once this work is committed the working tree is clean and the
		// assertion above would pass while checking nothing, so require the change set
		// to actually contain this spec's own files.
		own := exec.Command("git", "status", "--porcelain", "--", "pkg/initialize")
		own.Dir = repo
		mine, ownErr := own.Output()
		if ownErr != nil || strings.TrimSpace(string(mine)) == "" {
			t.Skip("this spec's own files are no longer uncommitted, so a working-tree comparison can no longer attribute anything; re-run this before committing")
		}
	})

	t.Run("the remoteless message is not masked", func(t *testing.T) {
		// Init must not paper over that message by rewriting, suppressing or
		// substituting for it in its own report. It holds none of its text.
		assertAbsentFromText(t, `(?i)(missing origin remote|baseline pull|remote baseline fetch|no cached baseline)`,
			"Init must neither rewrite, suppress nor substitute for the gate's own remoteless baseline_comparison message. Reproducing its text here is the first step toward replacing it.")
	})
}

// TestInit_ImplementsNoLocalProvenanceCache (SPEC-069 CLM-089).
//
// No local-provenance cache, no pack-sources record, and no lock-schema change. That
// half is ISSUE-055's, and this spec's obligation for it is a seam plus this absence.
func TestInit_ImplementsNoLocalProvenanceCache(t *testing.T) {
	assertAbsentFromCode(t, `(?i)(provenanceCache|packSources|pack-sources|localProvenance|sourceCache)`,
		"The local-provenance half is ISSUE-055's. The init source set must contain no cache, no pack-sources record and no lock-schema change.")

	// And nothing here touches the lock's own schema: init installs THROUGH the shipped
	// add path, which is what writes the lock.
	assertAbsentFromCode(t, `(?i)(LockEntry\{|WriteLockfile|lockfile\.Packs\s*\[)`,
		"Init writes no lock entry of its own. The shipped add path owns the lock, which is exactly why a portable ref is the only thing init will hand it.")
}

// TestInit_ImplementsNoCIDetectionOrBespokeGuidance (SPEC-069 CLM-085).
//
// ★ SPEC-070 EXPLICITLY DEFERS TO THIS TEST to keep doctor's guidance off init's CI
// steps, so it must remain a REAL assertion rather than a placeholder.
//
// Init never enumerates installed packs looking for a CI pack, never probes for a
// platform config file, and adds no guidance text beyond attributing the surfaced error
// to the CI step. The shipped errors already name what was missing and what IS
// available, which is precisely why init adds nothing.
func TestInit_ImplementsNoCIDetectionOrBespokeGuidance(t *testing.T) {
	code := initSourceCode(t)

	for _, step := range []string{"step_ci.go", "step_scaffold.go"} {
		body := code[findSourcePath(t, code, step)]

		t.Run(step+" enumerates no packs", func(t *testing.T) {
			for _, enumeration := range []string{"installedManifests", "loadInstalledPacks", "Manifest", "Engines"} {
				if strings.Contains(body, enumeration) {
					t.Fatalf("%s references %q; a recipe step that enumerated installed packs would be identifying \"the CI pack\", which is the detection REQ-017 forbids outright",
						step, enumeration)
				}
			}
		})

		t.Run(step+" probes no file on disk", func(t *testing.T) {
			for _, probe := range []string{"os.Stat", "os.ReadFile", "os.Open", "filepath.Walk", "os.ReadDir", "pathExists"} {
				if strings.Contains(body, probe) {
					t.Fatalf("%s calls %q; probing for a platform config file is detection, and the shipped resolve path is the only thing that may look for anything",
						step, probe)
				}
			}
		})

		t.Run(step+" adds no guidance beyond attribution", func(t *testing.T) {
			// The failure renderer is SHARED and adds an attribution prefix and nothing
			// else. A step-local failure message is where bespoke guidance would enter.
			if strings.Count(body, "failedRecipeStep") == 0 {
				t.Fatalf("%s does not surface failures through the shared attribution-only renderer", step)
			}
			for _, guidance := range []string{"try ", "you should", "we recommend", "it looks like", "probably", "make sure"} {
				if strings.Contains(strings.ToLower(body), guidance) {
					t.Fatalf("%s carries the guidance phrase %q; any further guidance belongs to `recipe apply`, not to init",
						step, guidance)
				}
			}
		})
	}
}

// nonImportStringLiterals returns every string literal in a file EXCEPT the ones in
// import declarations.
//
// The exclusion is structural rather than a prefix guess: an import path and a pack ref
// are the same SHAPE, so a scan that filtered by prefix would also hide a genuine pack
// name that happened to carry one.
func nonImportStringLiterals(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	imported := map[*ast.BasicLit]bool{}
	for _, spec := range parsed.Imports {
		imported[spec.Path] = true
	}

	literals := []string{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || imported[lit] {
			return true
		}
		literals = append(literals, lit.Value)
		return true
	})
	return literals
}

// TestInit_HoldsExactlyOneExecConstructionAndItTakesNoConsumerInput is the POSITIVE
// pin behind CLM-109's absence claim.
//
// ★ WHY A POSITIVE PIN RATHER THAN A FLAT BAN. CLM-109 reads "the source set contains
// no exec.Command construction", and the shipped source contains exactly one:
// step_git.go's `git init`, which REQ-006 mandates. A flat ban is therefore false as
// written, and the previous scan only covered the cmd/backstop half, so nothing caught
// the contradiction — the claim was true of the half that was scanned and false of the
// half that was not.
//
// THE HAZARD CLM-109 ACTUALLY GUARDS is running ARBITRARY PACK-SUPPLIED command strings
// outside the trusted-tool gate. `git init` is neither arbitrary nor pack-supplied: its
// argv is a compile-time constant with NO consumer input anywhere in it, and backstop's
// own use of git sits outside the allowlist by design — that allowlist governs
// PACK-DECLARED commands, which is why the whole pack-distribution path uses git the
// same way.
//
// So the invariant worth enforcing is not "zero" but "exactly one, in a named file,
// with no variable in its argv". That is strictly stronger than the flat ban over half
// the set: it would fail on a second construction anywhere, on this one moving, and —
// the property that actually matters — on this one starting to interpolate anything a
// consumer or a pack supplied.
func TestInit_HoldsExactlyOneExecConstructionAndItTakesNoConsumerInput(t *testing.T) {
	const sanctionedFile = "step_git.go"

	type execSite struct {
		file string
		fn   string
		call *ast.CallExpr
	}
	sites := []execSite{}

	for _, path := range initSourceSet(t) {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			ast.Inspect(fn.Body, func(inner ast.Node) bool {
				call, isCall := inner.(*ast.CallExpr)
				if !isCall {
					return true
				}
				selector, isSelector := call.Fun.(*ast.SelectorExpr)
				if !isSelector {
					return true
				}
				pkg, isIdent := selector.X.(*ast.Ident)
				if !isIdent || pkg.Name != "exec" {
					return true
				}
				if selector.Sel.Name == "Command" || selector.Sel.Name == "CommandContext" {
					sites = append(sites, execSite{file: filepath.Base(path), fn: fn.Name.Name, call: call})
				}
				return true
			})
			return true
		})
	}

	if len(sites) != 1 {
		rendered := []string{}
		for _, site := range sites {
			rendered = append(rendered, site.file+":"+site.fn)
		}
		t.Fatalf("the init source set holds %d exec constructions (%v), want EXACTLY ONE.\nEvery pack-declared command runs through the shared entrypoint prober's trusted-tool gate; a second construction here is a route around it.",
			len(sites), rendered)
	}
	if sites[0].file != sanctionedFile {
		t.Fatalf("the one exec construction lives in %s, want %s. Its sanction is specific to REQ-006's `git init`; moving it elsewhere moves it outside what was sanctioned.",
			sites[0].file, sanctionedFile)
	}

	// ★ THE PROPERTY THAT ACTUALLY MATTERS: no consumer or pack input reaches this argv.
	// Every argument must be a literal or a plain context — never a variable that could
	// carry a ref, a path fragment, or a pack-declared string.
	for i, arg := range sites[0].call.Args {
		switch typed := arg.(type) {
		case *ast.BasicLit:
			if typed.Kind != token.STRING {
				t.Fatalf("argument %d of the sanctioned exec construction is a non-string literal (%v)", i, typed.Kind)
			}
		case *ast.CallExpr:
			// context.Background() and the like: a constructed context carries no
			// command text. Anything else that is a call is suspect.
			selector, ok := typed.Fun.(*ast.SelectorExpr)
			if !ok {
				t.Fatalf("argument %d of the sanctioned exec construction is an unrecognized call", i)
			}
			if pkg, isIdent := selector.X.(*ast.Ident); !isIdent || pkg.Name != "context" {
				t.Fatalf("argument %d of the sanctioned exec construction calls something other than context.*; only a context and string literals may appear", i)
			}
		default:
			t.Fatalf("argument %d of the sanctioned exec construction is a %T rather than a string literal. Its whole sanction rests on the argv being a COMPILE-TIME CONSTANT with no consumer or pack input in it — a variable here is exactly the arbitrary-command hazard the trusted-tool gate exists to stop.",
				i, arg)
		}
	}
}
