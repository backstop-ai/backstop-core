package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteCheckRun(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("run code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "delivery inventory valid") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--root", t.TempDir()}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "read inventory") {
		t.Fatalf("missing code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"--unknown"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "flag provided") {
		t.Fatalf("flag code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--root", root, "--built-root", t.TempDir(), "--site-commit", ownerTestCommit, "--design-system-matrix"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "canonical-route") {
		t.Fatalf("invalid rendered site code=%d stderr=%q", code, stderr.String())
	}
}
