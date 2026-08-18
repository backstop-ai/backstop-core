package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// grep_convert_shape_test.go (PLAN-ISSUE-166, CLM-001/CLM-002/CLM-005): grep's
// stdout SHAPE is not a constant, and the pack-shipped grep->SARIF converts parse
// it. GNU grep OMITS the filename when the target is exactly ONE explicit file,
// emitting a 2-field "line:text" instead of "file:line:text"; the converts' old
// `NF >= 3` guard dropped every such line SILENTLY, at exit 0, so an absence probe
// reported zero matches whatever the file contained. These tests pin the two halves
// of the answer: the converts REFUSE what they cannot parse (they never guess), and
// the declarations force `-H -I` so there is nothing unparseable to refuse.
//
// PLATFORM HONESTY. The defect is a GNU-vs-BSD divergence and this file must be
// meaningful on both. The conversion legs feed SYNTHETIC stdin — the verbatim bytes
// GNU grep produced on the real Linux run — so they reproduce the defect exactly
// without a Linux host. The real-grep leg asserts ONLY platform-INVARIANT
// properties and RECORDS the divergent one with t.Log rather than asserting it.

// GNUTwoFieldGrepOutput is the VERBATIM stdout GNU grep emitted on the real Linux
// CI run, quoted from ISSUE-166 (lines 152-153 of the issue). It is the filename-
// less 2-field shape: "<line>:<text>". Both lines are real matches for the
// forbidden symbol; a convert that drops them reports a clean absence for a file
// that plainly contains the symbol. Stated ONCE here and fed to every convert, so
// a second spelling of the fixture cannot drift from this one.
//
// It is EXPORTED (the standard export_test.go idiom) so the installed-pack test in
// the external `engine_test` package feeds the released pack's convert these exact
// bytes. That package cannot live in `package engine` — it needs the production
// pack/lockfile readers, and `pkg/pack` imports `pkg/pack/engine`, so an in-package
// test importing it is an import cycle.
const GNUTwoFieldGrepOutput = `6:// "legacyProbeSymbol" appears here (even in a comment-adjacent identifier), the
8:func legacyProbeSymbol() string { return "should have been deleted" }
`

// bsdBinaryNoticeGrepOutput is real BSD grep stdout for a directory scan over a
// tree containing a binary file, measured 2026-08-18 against
// `grep (BSD grep, GNU compatible) 2.6.0-FreeBSD` (stderr was EMPTY — the notice
// really does land on STDOUT, interleaved with matches). The middle line is NOT a
// match and has no `file:line:` shape.
const bsdBinaryNoticeGrepOutput = `./one.go:1:legacyProbeSymbol here
Binary file ./blob.bin matches
./sub/two.go:1:legacyProbeSymbol two
`

// grepProbeSymbol is the forbidden SOURCE SYMBOL the contracts fixtures use — the
// identifier an absence contract forbids, not a credential. It is the same symbol
// the convert fixtures above carry, so the real-grep leg and the synthetic legs are
// talking about the same probe.
const grepProbeSymbol = "legacyProbeSymbol"

// discoverGrepConverts walks the repo for every grep->SARIF convert script the repo
// OWNS, rather than listing them. Listing is what let a third byte-identical copy
// (ts-proof-pack) sit unnoticed while the issue named only two; discovery covers a
// future copy the moment it is created, which is the operable form of the
// "keep them in sync" convention this lane declined to enforce textually.
//
// Excluded: `.backstop/` (gitignored INSTALL OUTPUT — the installed pack is fixed
// at its source repo and pulled in with `pack update`, never edited in place) and
// `.git/`.
func discoverGrepConverts(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".backstop", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != "to-sarif.sh" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "grep" {
			return nil // ast-grep converts parse JSON with explicit file fields; out of scope.
		}
		found = append(found, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking %s for grep converts: %v", root, walkErr)
	}
	sort.Strings(found)
	return found
}

// RunGrepConvert shells a convert script under /bin/sh with the given stdin and
// returns stdout, stderr and the exit status. Unlike the package's existing
// runConvertScript it does NOT fail on a non-zero exit: a non-zero exit is the very
// behavior CLM-001 asserts.
//
// Exported alongside GNUTwoFieldGrepOutput, and for the same reason: the installed
// go-contracts pack must be driven by the SAME runner as the in-repo copies, not by
// a second implementation that could drift from it.
func RunGrepConvert(t *testing.T, script string, stdin string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command("/bin/sh", script)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running convert %s: %v", script, err)
		}
		exitCode = exitErr.ExitCode()
	}
	return out.String(), errBuf.String(), exitCode
}

// sarifResultsOf parses a SARIF document and returns its results. An unparseable
// non-empty document is a failure — a convert that emits garbage is not "no
// findings".
func sarifResultsOf(t *testing.T, doc string) []struct {
	RuleID    string `json:"ruleId"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
} {
	t.Helper()
	var parsed struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(doc), &parsed); err != nil {
		t.Fatalf("convert stdout is not valid SARIF JSON: %v\nstdout was: %q", err, doc)
	}
	if parsed.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want 2.1.0", parsed.Version)
	}
	if len(parsed.Runs) != 1 {
		t.Fatalf("SARIF runs length = %d, want 1", len(parsed.Runs))
	}
	return parsed.Runs[0].Results
}

// assertNoFabricatedFinding asserts a REFUSING convert emitted no SARIF result:
// either nothing at all on stdout, or a well-formed document with an empty results
// array. What it must never do is invent one. Section 2 of the plan measured why
// this matters: a "robust" 2-field parser turns `6:42: text` into a finding at a
// file literally named `6`, trading a silent false negative for a silent false
// positive in the same security-relevant absence probe.
func assertNoFabricatedFinding(t *testing.T, stdout string) {
	t.Helper()
	if strings.TrimSpace(stdout) == "" {
		return
	}
	if results := sarifResultsOf(t, stdout); len(results) != 0 {
		t.Errorf("refusing convert emitted %d SARIF result(s); a refusal must fabricate nothing: %s",
			len(results), stdout)
	}
}

// TestGrepConvert_RefusesUnparseableGrepLineLoudly (CLM-001) asserts that every
// grep->SARIF convert this repo OWNS refuses a stdin line it cannot parse as
// `<file>:<line>:<text>` — non-zero exit plus a stderr diagnostic naming the
// offending line and the expected shape — instead of dropping it silently.
//
// Dropping a line from an absence probe at exit 0 with no diagnostic is the exact
// silent-vacuous-green shape this repo forbids: the gate reports "symbol absent"
// for a file that contains it. The fix is to REFUSE, never to guess (see
// assertNoFabricatedFinding for why guessing is worse than the bug).
func TestGrepConvert_RefusesUnparseableGrepLineLoudly(t *testing.T) {
	root := repoRoot(t)
	converts := discoverGrepConverts(t, root)
	if len(converts) == 0 {
		t.Fatal("discovered ZERO grep convert scripts — the discovery walk is broken, " +
			"and a test that checks nothing would pass vacuously")
	}
	// At authoring the repo owns five (packs/contracts, traceability-pack,
	// ts-proof-pack, and the two hermetic-remote fixture packs). Assert the floor
	// rather than the exact list so a NEW copy is covered on creation instead of
	// silently escaping the convention.
	const wantAtLeast = 5
	if len(converts) < wantAtLeast {
		t.Errorf("discovered %d grep converts, want at least %d — a copy went missing "+
			"or the walk narrowed", len(converts), wantAtLeast)
	}
	for _, c := range converts {
		rel, _ := filepath.Rel(root, c)
		t.Logf("discovered grep convert: %s", rel)
	}

	cases := []struct {
		name string
		// stdin is the raw grep stdout fed to the convert.
		stdin string
		// mustName is a distinctive substring of the OFFENDING line; the diagnostic
		// has to identify WHICH line failed, not merely that one did.
		//
		// The convert refuses at the FIRST unparseable line and stops, so this is a
		// substring of that line — not of a later one. Stopping early is deliberate:
		// the first bad line already proves the input shape is wrong, and continuing
		// would mean deciding what to do with the rest, which is the "skip it and
		// keep going" drop this fix removes.
		mustName string
		// mustNameRemedy, when set, is additionally required in stderr.
		mustNameRemedy string
	}{
		{
			// The real GNU CI bytes. Two genuine matches for the forbidden symbol,
			// both filename-less because the grep target was a single explicit file.
			name:     "gnu_two_field_single_file_shape",
			stdin:    GNUTwoFieldGrepOutput,
			mustName: "comment-adjacent identifier",
		},
		{
			// grep's OWN non-match line, interleaved with real matches exactly as BSD
			// grep emits it on STDOUT (measured; stderr was empty).
			//
			// THIS REFUSAL IS DELIBERATE, and it is why every declaration passes -I.
			// The correct fix is to stop the line being EMITTED, not to teach the
			// convert grep's notice wording: that wording and even its stream differ
			// between BSD and GNU, and the GNU side is not measurable from darwin.
			// Matching text we cannot observe is precisely the guessing that `-H`/`-I`
			// exist to avoid. So the convert refuses, and the diagnostic must name -I
			// as the remedy — whoever trips this is a pack author who omitted a flag,
			// and the message is their only clue.
			//
			// NOTE the absence of any assertion that the two REAL matches still
			// convert. Refusal means refusal: a test expecting partial output would
			// license a "skip the weird line and keep going" implementation, which is
			// the silent drop returning under a new name.
			name:           "grep_own_binary_file_notice_on_stdout",
			stdin:          bsdBinaryNoticeGrepOutput,
			mustName:       "Binary file ./blob.bin matches",
			mustNameRemedy: "-I",
		},
	}

	for _, c := range converts {
		rel, _ := filepath.Rel(root, c)
		for _, tc := range cases {
			t.Run(rel+"/"+tc.name, func(t *testing.T) {
				stdout, stderr, code := RunGrepConvert(t, c, tc.stdin)
				if code == 0 {
					t.Errorf("convert exited 0 on unparseable input — this is the SILENT "+
						"DROP the fix removes.\nstdout: %s\nstderr: %s", stdout, stderr)
				}
				if !strings.Contains(stderr, tc.mustName) {
					t.Errorf("stderr must NAME the offending line (want substring %q) so an "+
						"operator knows WHICH line and WHY without re-running.\nstderr: %s",
						tc.mustName, stderr)
				}
				if !strings.Contains(stderr, "<file>:<line>:<text>") {
					t.Errorf("stderr must state the expected <file>:<line>:<text> shape.\nstderr: %s",
						stderr)
				}
				if tc.mustNameRemedy != "" && !strings.Contains(stderr, tc.mustNameRemedy) {
					t.Errorf("stderr must name %q as the remedy — the operator is a pack author "+
						"missing a flag.\nstderr: %s", tc.mustNameRemedy, stderr)
				}
				assertNoFabricatedFinding(t, stdout)
			})
		}
	}
}

// TestGrepConvert_ThreeFieldMatchesAndEmptyInputUnchanged (CLM-002) is the
// PRESERVATION leg. It must be green BEFORE and AFTER the refusal lands, and it is
// what stops the refusal being implemented by refusing everything: a zero-match run
// is the ORDINARY case for an absence probe and stays a clean pass.
//
// The two hermetic-remote converts emit their own ruleIds on purpose, so these
// assertions are on result COUNT and LOCATION only — never on a ruleId literal.
func TestGrepConvert_ThreeFieldMatchesAndEmptyInputUnchanged(t *testing.T) {
	root := repoRoot(t)
	converts := discoverGrepConverts(t, root)
	if len(converts) == 0 {
		t.Fatal("discovered ZERO grep convert scripts — nothing would be asserted")
	}

	for _, c := range converts {
		rel, _ := filepath.Rel(root, c)

		t.Run(rel+"/happy_path_three_field", func(t *testing.T) {
			stdout, stderr, code := RunGrepConvert(t, c,
				"a.go:8:func legacyProbeSymbol() string { return \"x\" }\n")
			if code != 0 {
				t.Fatalf("well-formed input must convert cleanly; exit=%d stderr=%s", code, stderr)
			}
			results := sarifResultsOf(t, stdout)
			if len(results) != 1 {
				t.Fatalf("want exactly 1 SARIF result, got %d: %s", len(results), stdout)
			}
			loc := results[0].Locations[0].PhysicalLocation
			if loc.ArtifactLocation.URI != "a.go" {
				t.Errorf("artifactLocation.uri = %q, want %q", loc.ArtifactLocation.URI, "a.go")
			}
			if loc.Region.StartLine != 8 {
				t.Errorf("region.startLine = %d, want 8 (grep's line numbers are 1-indexed)",
					loc.Region.StartLine)
			}
			if !strings.Contains(results[0].Message.Text, "legacyProbeSymbol") {
				t.Errorf("message must carry the matched text, got %q", results[0].Message.Text)
			}
		})

		t.Run(rel+"/colon_in_match_text", func(t *testing.T) {
			// Only the FIRST TWO colons are structural. A naive field re-join under
			// -F: truncates the match text at the next colon, so this case is what
			// keeps the prefix-strip recovery honest.
			stdout, stderr, code := RunGrepConvert(t, c,
				"a.go:8:const x = \"http://y\"\n")
			if code != 0 {
				t.Fatalf("colon-bearing match text must convert cleanly; exit=%d stderr=%s", code, stderr)
			}
			results := sarifResultsOf(t, stdout)
			if len(results) != 1 {
				t.Fatalf("want exactly 1 SARIF result, got %d: %s", len(results), stdout)
			}
			if !strings.Contains(results[0].Message.Text, "http://y") {
				t.Errorf("match text after the second colon must survive intact, got %q",
					results[0].Message.Text)
			}
			loc := results[0].Locations[0].PhysicalLocation
			if loc.ArtifactLocation.URI != "a.go" || loc.Region.StartLine != 8 {
				t.Errorf("location = %s:%d, want a.go:8",
					loc.ArtifactLocation.URI, loc.Region.StartLine)
			}
		})

		t.Run(rel+"/empty_stdin_is_a_clean_pass", func(t *testing.T) {
			// ZERO MATCHES IS THE ORDINARY CASE for an absence probe. Getting this
			// backwards turns every clean run into a red. Empty STDOUT is also a
			// failure: a downstream SARIF parse of an empty payload reads as a broken
			// pack, not as a clean run.
			stdout, stderr, code := RunGrepConvert(t, c, "")
			if code != 0 {
				t.Fatalf("empty stdin must exit 0 (zero matches is a clean pass); exit=%d stderr=%s",
					code, stderr)
			}
			results := sarifResultsOf(t, stdout)
			if len(results) != 0 {
				t.Errorf("empty stdin must yield an EMPTY results array, got %d", len(results))
			}
		})
	}
}

// grepLineShape reports whether a line has the structural `<file>:<digits>:` prefix
// that every convert parses. It is the same predicate the converts apply, restated
// in Go so the test does not have to trust the scripts to describe themselves.
func grepLineShape(line string) bool {
	first := strings.Index(line, ":")
	if first <= 0 {
		return false
	}
	rest := line[first+1:]
	second := strings.Index(rest, ":")
	if second <= 0 {
		return false
	}
	for _, r := range rest[:second] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// TestRealGrep_FilenameHeaderWithDashHIsPlatformInvariant (CLM-005) is a
// CHARACTERIZATION test against the REAL grep binary. It asserts only what is true
// on GNU grep and BSD grep alike, and RECORDS the one property that legitimately
// differs.
//
// WHY THE RESTRAINT MATTERS. darwin's grep is DIFFERENT here, not WRONG. Asserting
// darwin's single-file shape would be a platform lock-in wearing a correctness
// check's clothes — and it would go red on the very CI runner this lane exists to
// fix.
func TestRealGrep_FilenameHeaderWithDashHIsPlatformInvariant(t *testing.T) {
	// The shell in this environment may define a `grep` FUNCTION wrapping a
	// different tool entirely; exec.Command resolves the real binary on PATH. Log
	// which one, so a reading of this test describes the tool the gate actually runs.
	grepBin, err := exec.LookPath("grep")
	if err != nil {
		t.Fatalf("real grep is required for this characterization (no t.Skip): %v", err)
	}
	t.Logf("resolved grep binary: %s", grepBin)
	if ver, verErr := exec.Command(grepBin, "--version").Output(); verErr == nil {
		t.Logf("grep version: %s", strings.SplitN(strings.TrimSpace(string(ver)), "\n", 2)[0])
	}

	dir := t.TempDir()
	if mkErr := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); mkErr != nil {
		t.Fatalf("mkdir sub: %v", mkErr)
	}
	oneGo := filepath.Join(dir, "one.go")
	twoGo := filepath.Join(dir, "sub", "two.go")
	writeFile := func(path string, data []byte) {
		if wErr := os.WriteFile(path, data, 0o600); wErr != nil {
			t.Fatalf("writing %s: %v", path, wErr)
		}
	}
	writeFile(oneGo, []byte("package p\n"+grepProbeSymbol+" here\n"))
	writeFile(twoGo, []byte("package q\n"+grepProbeSymbol+" two\n"))
	// THE BINARY FILE IS REQUIRED, NOT DECORATION: it is what lets the -I legs
	// below be able to FAIL. A NUL and a high byte are what make grep classify the
	// file as binary.
	writeFile(filepath.Join(dir, "blob.bin"),
		append([]byte{'M', 'Z', 0x00, 0xff, 0xfe}, []byte(grepProbeSymbol+" tail\x00\n")...))

	runGrepArgs := func(args ...string) []string {
		cmd := exec.Command(grepBin, args...)
		cmd.Dir = dir
		var out, errBuf bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errBuf
		_ = cmd.Run() // grep exits non-zero on no-match; stdout is the contract.
		if errBuf.Len() > 0 {
			t.Logf("grep %v wrote to stderr: %q", args, errBuf.String())
		}
		var lines []string
		for _, l := range strings.Split(out.String(), "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
		return lines
	}

	// ── LEG 1: WITH -H, EVERY MATCH LINE CARRIES A FILENAME PREFIX ──────────────
	// This is the invariant the entire fix rests on. -I accompanies it so that the
	// only lines on stdout ARE matches (see leg 3).
	for _, tc := range []struct {
		name   string
		target []string
	}{
		{"single_explicit_file", []string{"one.go"}},
		{"two_explicit_files", []string{"one.go", "sub/two.go"}},
		{"directory_target", []string{"."}},
	} {
		t.Run("dash_H_always_prefixes/"+tc.name, func(t *testing.T) {
			args := append([]string{"-rn", "-H", "-I", "-e", grepProbeSymbol}, tc.target...)
			lines := runGrepArgs(args...)
			if len(lines) == 0 {
				t.Fatalf("expected matches for %v, got none", tc.target)
			}
			for _, l := range lines {
				if !grepLineShape(l) {
					t.Errorf("line %q does not carry the <file>:<line>: prefix -H guarantees", l)
				}
				if strings.HasPrefix(l, ":") {
					t.Errorf("line %q has an EMPTY filename field", l)
				}
			}
			t.Logf("%s with -H -I: %v", tc.name, lines)
		})
	}

	// ── LEG 2: -H IS IDEMPOTENT WHERE IT ALREADY WORKED ─────────────────────────
	// This is what proves adding -H to the declarations and helpers is not a
	// behavior change for the paths that already work — only a pin on the one that
	// diverges. Directory walk order is unspecified, so compare sorted.
	sortedEqual := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		as := append([]string(nil), a...)
		bs := append([]string(nil), b...)
		sort.Strings(as)
		sort.Strings(bs)
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
		return true
	}
	for _, tc := range []struct {
		name   string
		target []string
	}{
		{"two_explicit_files", []string{"one.go", "sub/two.go"}},
		{"directory_target", []string{"."}},
	} {
		t.Run("dash_H_is_idempotent/"+tc.name, func(t *testing.T) {
			without := runGrepArgs(append([]string{"-rn", "-e", grepProbeSymbol}, tc.target...)...)
			with := runGrepArgs(append([]string{"-rn", "-H", "-e", grepProbeSymbol}, tc.target...)...)
			if len(without) == 0 {
				t.Fatalf("expected matches for %v, got none", tc.target)
			}
			if !sortedEqual(without, with) {
				t.Errorf("-H changed output for a target that already worked.\nwithout: %v\nwith:    %v",
					without, with)
			}
		})
	}

	// ── LEG 3: WITH -I, grep EMITS NO NON-MATCH LINE AT ALL ─────────────────────
	// This is the assertion that keeps the binary-notice trap closed, and it is
	// platform-neutral BY CONSTRUCTION: it asserts the ABSENCE of a notice, not its
	// wording. It therefore holds whether the host grep would have written that
	// notice to stdout (BSD — measured here) or to stderr (GNU — reported, but not
	// measurable from darwin). Asserting the wording is exactly the guessing that
	// forcing the flags exists to avoid.
	t.Run("dash_I_leaves_only_match_lines_on_stdout", func(t *testing.T) {
		withoutI := runGrepArgs("-rn", "-H", "-e", grepProbeSymbol, ".")
		withI := runGrepArgs("-rn", "-H", "-I", "-e", grepProbeSymbol, ".")

		var unparseable []string
		for _, l := range withoutI {
			if !grepLineShape(l) {
				unparseable = append(unparseable, l)
			}
		}
		// RECORDED, not asserted: whether this host's grep puts the notice on stdout
		// at all. On BSD 2.6.0-FreeBSD it does; the point of -I is that it stops
		// mattering.
		t.Logf("without -I, %d non-match line(s) landed on stdout: %v",
			len(unparseable), unparseable)

		if len(withI) == 0 {
			t.Fatal("expected matches with -I, got none")
		}
		for _, l := range withI {
			if !grepLineShape(l) {
				t.Errorf("with -I, stdout still carries a non-match line %q — the convert "+
					"would be asked to parse it", l)
			}
		}
	})

	// ── LEG 4: -I IS RESULT-NEUTRAL — IT REMOVES NO EVIDENCE ────────────────────
	// The property that justifies adding -I everywhere: it suppresses grep's
	// informational notice and NOTHING else. Measured 2026-08-18 through the
	// then-unfixed pack convert, the directory scan yielded 2 SARIF results with -I
	// and 2 without — the binary file's content never became a finding either way.
	//
	// ★ WHY THIS LEG COMPARES MATCH LINES RATHER THAN PIPING BOTH STREAMS THROUGH
	// THE CONVERT. Once the convert REFUSES unparseable input (CLM-001), the
	// without-`-I` stream on BSD contains grep's `Binary file …` notice and is
	// therefore refused outright — 0 results, by design. Piping both through the
	// convert would then assert 0 == 2 and fail, and the only way to "fix" it would
	// be to weaken the refusal. So the neutrality property is asserted where it
	// actually lives: on the MATCH SET grep produces. Same claim, and it survives
	// the fix instead of contradicting it.
	t.Run("dash_I_is_result_neutral", func(t *testing.T) {
		matchLines := func(lines []string) []string {
			var out []string
			for _, l := range lines {
				if grepLineShape(l) {
					out = append(out, l)
				}
			}
			return out
		}
		withoutI := matchLines(runGrepArgs("-rn", "-H", "-e", grepProbeSymbol, "."))
		withI := matchLines(runGrepArgs("-rn", "-H", "-I", "-e", grepProbeSymbol, "."))
		if len(withI) == 0 {
			t.Fatal("expected matches with -I, got none")
		}
		if !sortedEqual(withoutI, withI) {
			t.Errorf("-I removed EVIDENCE, not just the notice.\nwithout -I: %v\nwith -I:    %v",
				withoutI, withI)
		}

		// And the convert turns exactly those match lines into exactly that many
		// SARIF results — so "same matches" really does mean "same findings".
		convert := filepath.Join(repoRoot(t), durablePackRel, "grep", "to-sarif.sh")
		stdout, stderr, code := RunGrepConvert(t, convert, strings.Join(withI, "\n")+"\n")
		if code != 0 {
			t.Fatalf("the -I stream must convert cleanly; exit=%d stderr=%s", code, stderr)
		}
		if got := len(sarifResultsOf(t, stdout)); got != len(withI) {
			t.Errorf("convert produced %d results for %d match lines", got, len(withI))
		}
	})

	// ── RECORDED, NEVER ASSERTED: the single-file shape without -H ───────────────
	// GNU grep OMITS the filename when the target is exactly one explicit file and
	// -H is absent; BSD grep 2.6.0-FreeBSD PRINTS it under -rn. Both are correct.
	// This log is the divergence itself, captured on whichever host runs the test.
	// DO NOT "fix" it into an assertion — either polarity would be a platform
	// lock-in, and it would red on the other platform.
	single := runGrepArgs("-rn", "-e", grepProbeSymbol, "one.go")
	shape := "2-field (line:text) — the GNU shape this lane fixes"
	if len(single) > 0 && grepLineShape(single[0]) {
		shape = "3-field (file:line:text) — the BSD shape"
	}
	t.Logf("RECORDED (not asserted): single explicit file, -rn WITHOUT -H → %s: %v",
		shape, single)
}
