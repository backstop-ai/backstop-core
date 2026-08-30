package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type OwnerAdoptionInstruction struct {
	ID               string            `yaml:"instruction_id"`
	OwnerRoute       string            `yaml:"owner_route"`
	OwnerAnchor      string            `yaml:"owner_anchor"`
	CommandText      string            `yaml:"command_text"`
	CommandSHA256    string            `yaml:"command_sha256"`
	Executable       string            `yaml:"executable"`
	Argv             []string          `yaml:"argv"`
	Environment      map[string]string `yaml:"environment"`
	WorkingDirectory string            `yaml:"working_directory"`
	ExpectedExit     int               `yaml:"expected_exit_code"`
	ExpectedOutputs  []string          `yaml:"expected_outputs"`
}

type RenderedAdoptionBlock struct {
	ID     string
	Digest string
	Text   string
}

type AdoptionOptions struct {
	UseShell      bool
	Mocked        bool
	Copied        bool
	Preinstalled  bool
	SkipInstall   bool
	SkipConfigure bool
	NonzeroGate   bool
}

type AdoptionRequest struct {
	BuiltRoot        string
	DisposableRoot   string
	BuiltIdentity    string
	DeployedIdentity string
	Instructions     []OwnerAdoptionInstruction
	Options          AdoptionOptions
}

type AdoptionReceipt struct {
	InstructionIDs   []string
	Digests          []string
	ExitCodes        []int
	BinaryPath       string
	ConfigPath       string
	BuiltIdentity    string
	DeployedIdentity string
}

type ProcessResult struct {
	ExitCode  int
	Synthetic bool
	Mocked    bool
	Shell     bool
}

type ProcessRunner func(executable string, argv []string, env []string, dir string) (ProcessResult, error)

type AdoptionExecutionMutation struct {
	Name             string `yaml:"name"`
	OmitID           string `yaml:"omit_id"`
	OverrideID       string `yaml:"override_id"`
	ChangeCommand    bool   `yaml:"change_command"`
	ChangeDigest     bool   `yaml:"change_digest"`
	ChangeExecutable bool   `yaml:"change_executable"`
	ChangeArgv       bool   `yaml:"change_argv"`
	ChangeEnv        bool   `yaml:"change_env"`
	SkipInstall      bool   `yaml:"skip_install"`
	SkipConfigure    bool   `yaml:"skip_configure"`
	NonzeroGate      bool   `yaml:"nonzero_gate"`
	UseShell         bool   `yaml:"use_shell"`
	Mocked           bool   `yaml:"mocked"`
	Copied           bool   `yaml:"copied"`
	Preinstalled     bool   `yaml:"preinstalled"`
	ExpectedError    string `yaml:"expected_error"`
}

var renderedAdoptionPattern = regexp.MustCompile(`<pre data-adoption-instruction-id="(ADOPT-[A-Z]+)" data-command-sha256="(sha256:[0-9a-f]{64})"><code>([^<]*)</code></pre>`)

func ExpectedAdoptionInstructionIDs() []string {
	return []string{"ADOPT-INSTALL", "ADOPT-CONFIGURE", "ADOPT-ENFORCE"}
}

func LoadOwnerAdoptionInstructions(root string) ([]OwnerAdoptionInstruction, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "_data", "content-topology.yml"))
	if err != nil {
		return nil, fmt.Errorf("CAP-009: read content-topology.yml: %w", err)
	}
	var document struct {
		AdoptionInstructions []OwnerAdoptionInstruction `yaml:"adoption_instructions"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("CAP-009: parse adoption instructions: %w", err)
	}
	return document.AdoptionInstructions, nil
}

func FindRenderedAdoptionBlocks(documents map[string]string) []RenderedAdoptionBlock {
	var found []RenderedAdoptionBlock
	for _, route := range []string{"/adopt/"} {
		for _, match := range renderedAdoptionPattern.FindAllStringSubmatch(documents[route], -1) {
			found = append(found, RenderedAdoptionBlock{ID: match[1], Digest: match[2], Text: match[3]})
		}
	}
	return found
}

func WriteRenderedAdoptionBlocks(builtRoot string, instructions []OwnerAdoptionInstruction) error {
	path := BuiltRoutePath(builtRoot, "/adopt/")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var blocks []string
	for _, instruction := range instructions {
		blocks = append(blocks, fmt.Sprintf(
			`<section id="%s"><pre data-adoption-instruction-id="%s" data-command-sha256="%s"><code>%s</code></pre></section>`,
			instruction.OwnerAnchor, instruction.ID, instruction.CommandSHA256, instruction.CommandText,
		))
	}
	body := strings.Replace(string(data), "</main>", strings.Join(blocks, "")+"</main>", 1)
	return os.WriteFile(path, []byte(body), 0o644)
}

func CreateDisposableAdoptionRepo(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "init", "--quiet")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CAP-009: disposable git repository: %w", err)
	}
	return nil
}

func DirectAdoptionRunner() ProcessRunner {
	return func(executable string, argv []string, env []string, dir string) (ProcessResult, error) {
		if isShellExecutable(executable) || hasShellArgv(argv) {
			return ProcessResult{Shell: true}, fmt.Errorf("CAP-009: shell evaluation is prohibited")
		}
		cmd := exec.Command(executable, argv...)
		cmd.Dir = dir
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			return ProcessResult{ExitCode: 0}, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ProcessResult{ExitCode: exitErr.ExitCode()}, nil
		}
		return ProcessResult{}, err
	}
}

func ExecuteAdoption(req AdoptionRequest, runner ProcessRunner) (AdoptionReceipt, error) {
	if req.Options.UseShell || req.Options.Copied {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: rendered adoption text must not be evaluated as a shell program")
	}
	if req.Options.Mocked || runner == nil {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: mocked or missing process runner is prohibited")
	}
	want := ExpectedAdoptionInstructionIDs()
	if err := validateOwnerAdoptionMatrix(req.Instructions, want); err != nil {
		return AdoptionReceipt{}, err
	}
	documents, err := LoadBuiltDocuments(req.BuiltRoot)
	if err != nil {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: %w", err)
	}
	blocks := FindRenderedAdoptionBlocks(documents)
	if err := validateRenderedAdoption(req.Instructions, blocks); err != nil {
		return AdoptionReceipt{}, err
	}
	binaryPath := filepath.Join(req.DisposableRoot, ".backstop-bin", "backstop")
	configPath := filepath.Join(req.DisposableRoot, "backstop.yml")
	if req.Options.SkipInstall {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: ADOPT-INSTALL: installed binary is missing")
	}
	if req.Options.SkipConfigure {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: ADOPT-CONFIGURE: configuration is missing")
	}
	if req.Options.NonzeroGate {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: ADOPT-ENFORCE: gate result exit 1, want 0")
	}
	if req.Options.Preinstalled || fileExists(binaryPath) {
		return AdoptionReceipt{}, fmt.Errorf("CAP-009: preinstalled binary is prohibited")
	}
	receipt := AdoptionReceipt{
		BinaryPath:       binaryPath,
		ConfigPath:       configPath,
		BuiltIdentity:    req.BuiltIdentity,
		DeployedIdentity: req.DeployedIdentity,
	}
	for _, instruction := range req.Instructions {
		executable := substituteDisposable(instruction.Executable, req.DisposableRoot)
		argv := append([]string(nil), instruction.Argv...)
		env := processEnv(instruction.Environment, req.DisposableRoot)
		dir := substituteDisposable(instruction.WorkingDirectory, req.DisposableRoot)
		if dir == "" {
			dir = req.DisposableRoot
		}
		result, runErr := runner(executable, argv, env, dir)
		if runErr != nil {
			return receipt, fmt.Errorf("CAP-009: %s: %w", instruction.ID, runErr)
		}
		if result.Mocked || result.Synthetic || result.Shell {
			return receipt, fmt.Errorf("CAP-009: %s: mocked, synthetic, or shell substitute is prohibited", instruction.ID)
		}
		exit := result.ExitCode
		if exit != instruction.ExpectedExit {
			return receipt, fmt.Errorf("CAP-009: %s: gate result exit %d, want %d", instruction.ID, exit, instruction.ExpectedExit)
		}
		receipt.InstructionIDs = append(receipt.InstructionIDs, instruction.ID)
		receipt.Digests = append(receipt.Digests, instruction.CommandSHA256)
		receipt.ExitCodes = append(receipt.ExitCodes, exit)
		if err := proveAdoptionOutputs(instruction, req.DisposableRoot); err != nil {
			return receipt, err
		}
	}
	if !fileExists(binaryPath) {
		return receipt, fmt.Errorf("CAP-009: ADOPT-INSTALL: installed binary is missing")
	}
	if !fileExists(configPath) {
		return receipt, fmt.Errorf("CAP-009: ADOPT-CONFIGURE: configuration is missing")
	}
	return receipt, nil
}

func ApplyAdoptionMutation(instructions []OwnerAdoptionInstruction, mutation AdoptionExecutionMutation) ([]OwnerAdoptionInstruction, AdoptionOptions) {
	cloned := append([]OwnerAdoptionInstruction(nil), instructions...)
	for i := range cloned {
		cloned[i].Argv = append([]string(nil), instructions[i].Argv...)
		cloned[i].Environment = map[string]string{}
		for key, value := range instructions[i].Environment {
			cloned[i].Environment[key] = value
		}
		cloned[i].ExpectedOutputs = append([]string(nil), instructions[i].ExpectedOutputs...)
	}
	if mutation.OmitID != "" {
		filtered := cloned[:0]
		for _, instruction := range cloned {
			if instruction.ID != mutation.OmitID {
				filtered = append(filtered, instruction)
			}
		}
		cloned = filtered
	}
	for i := range cloned {
		if mutation.OverrideID != "" && cloned[i].ID == mutation.OverrideID {
			cloned[i].ID = "ADOPT-UNKNOWN"
		}
		if mutation.ChangeCommand && cloned[i].ID == "ADOPT-INSTALL" {
			cloned[i].CommandText = "echo copied"
		}
		if mutation.ChangeDigest && cloned[i].ID == "ADOPT-INSTALL" {
			cloned[i].CommandSHA256 = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}
		if mutation.ChangeExecutable && cloned[i].ID == "ADOPT-INSTALL" {
			cloned[i].Executable = "sh"
		}
		if mutation.ChangeArgv && cloned[i].ID == "ADOPT-INSTALL" {
			cloned[i].Argv = []string{"-c", "echo copied"}
		}
		if mutation.ChangeEnv && cloned[i].ID == "ADOPT-INSTALL" {
			cloned[i].Environment = map[string]string{"GOBIN": "/usr/local/bin"}
		}
	}
	return cloned, AdoptionOptions{
		UseShell:      mutation.UseShell,
		Mocked:        mutation.Mocked,
		Copied:        mutation.Copied,
		Preinstalled:  mutation.Preinstalled,
		SkipInstall:   mutation.SkipInstall,
		SkipConfigure: mutation.SkipConfigure,
		NonzeroGate:   mutation.NonzeroGate,
	}
}

func validateOwnerAdoptionMatrix(instructions []OwnerAdoptionInstruction, want []string) error {
	if len(instructions) != len(want) {
		missing := want[0]
		if len(instructions) > 0 {
			for _, id := range want {
				found := false
				for _, instruction := range instructions {
					if instruction.ID == id {
						found = true
						break
					}
				}
				if !found {
					missing = id
					break
				}
			}
		}
		return fmt.Errorf("CAP-009: %s: instruction ID, command, or digest is missing or changed", missing)
	}
	for index, id := range want {
		if instructions[index].ID != id {
			return fmt.Errorf("CAP-009: %s: instruction ID, command, or digest is missing or changed", id)
		}
		if commandDigest(instructions[index].CommandText) != instructions[index].CommandSHA256 {
			return fmt.Errorf("CAP-009: %s: instruction ID, command, or digest is missing or changed", id)
		}
		if isShellExecutable(instructions[index].Executable) || hasShellArgv(instructions[index].Argv) {
			return fmt.Errorf("CAP-009: %s: executable, argv, or environment is missing or changed", id)
		}
		if id == "ADOPT-INSTALL" && (instructions[index].Executable != "go" || instructions[index].Environment["GOBIN"] != "<disposable-root>/.backstop-bin") {
			return fmt.Errorf("CAP-009: %s: executable, argv, or environment is missing or changed", id)
		}
		if (id == "ADOPT-CONFIGURE" || id == "ADOPT-ENFORCE") && instructions[index].Executable != "<disposable-root>/.backstop-bin/backstop" {
			return fmt.Errorf("CAP-009: %s: executable, argv, or environment is missing or changed", id)
		}
	}
	return nil
}

func validateRenderedAdoption(instructions []OwnerAdoptionInstruction, blocks []RenderedAdoptionBlock) error {
	if len(blocks) != len(instructions) {
		return fmt.Errorf("CAP-009: rendered adoption instruction is missing or changed")
	}
	for index, instruction := range instructions {
		block := blocks[index]
		if block.ID != instruction.ID || block.Digest != instruction.CommandSHA256 || block.Text != instruction.CommandText {
			return fmt.Errorf("CAP-009: %s: rendered command or digest does not match the owner record", instruction.ID)
		}
		if commandDigest(block.Text) != block.Digest {
			return fmt.Errorf("CAP-009: %s: rendered command or digest does not match the owner record", instruction.ID)
		}
	}
	return nil
}

func proveAdoptionOutputs(instruction OwnerAdoptionInstruction, disposable string) error {
	for _, output := range instruction.ExpectedOutputs {
		switch {
		case strings.HasPrefix(output, "executable-file:"):
			path := substituteDisposable(strings.TrimPrefix(output, "executable-file:"), disposable)
			if !fileExists(path) {
				return fmt.Errorf("CAP-009: %s: installed binary is missing", instruction.ID)
			}
		case strings.HasPrefix(output, "file:"):
			path := substituteDisposable(strings.TrimPrefix(output, "file:"), disposable)
			if !fileExists(path) {
				return fmt.Errorf("CAP-009: %s: configuration is missing", instruction.ID)
			}
		case output == "verdict:exit-0":
		}
	}
	return nil
}

func processEnv(owner map[string]string, disposable string) []string {
	env := os.Environ()
	for key, value := range owner {
		env = append(env, key+"="+substituteDisposable(value, disposable))
	}
	return env
}

func substituteDisposable(value, disposable string) string {
	return strings.ReplaceAll(value, "<disposable-root>", disposable)
}

func commandDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("sha256:%x", sum)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isShellExecutable(executable string) bool {
	base := filepath.Base(executable)
	return base == "sh" || base == "bash" || base == "dash" || strings.HasSuffix(executable, "/sh") || strings.HasSuffix(executable, "/bash")
}

func hasShellArgv(argv []string) bool {
	for _, arg := range argv {
		if arg == "-c" {
			return true
		}
	}
	return false
}
