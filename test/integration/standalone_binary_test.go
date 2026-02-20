package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStandaloneBinaryVersionAndCommandsWorkOutsideRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("standalone binary copy/exec test is unix-focused")
	}
	goModPathBytes, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	goModPath := strings.TrimSpace(string(goModPathBytes))
	if goModPath == "" {
		t.Fatalf("go env GOMOD returned empty")
	}
	repoRoot := filepath.Dir(goModPath)

	versionBytes, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	expectedVersion := strings.TrimSpace(string(versionBytes))
	if expectedVersion == "" {
		t.Fatal("VERSION file is empty")
	}

	buildDir := t.TempDir()
	binaryPath := filepath.Join(buildDir, "namelens")

	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=integration-test -X main.buildDate=integration-test", expectedVersion)
	build := exec.Command("go", "build", "-tags", "sysprims_shared", "-ldflags", ldflags, "-o", binaryPath, "./cmd/namelens")
	build.Dir = repoRoot
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, string(out))
	}

	outside := t.TempDir()
	copiedBinary := filepath.Join(outside, "namelens")

	// Use a direct file copy to avoid relying on platform-specific tools.
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read built binary: %v", err)
	}
	if err := os.WriteFile(copiedBinary, data, 0o755); err != nil {
		t.Fatalf("write copied binary: %v", err)
	}

	cmdEnv := append(os.Environ(),
		"HOME="+outside,
		"XDG_CONFIG_HOME="+filepath.Join(outside, "xdg-config"),
		"XDG_CACHE_HOME="+filepath.Join(outside, "xdg-cache"),
		"XDG_DATA_HOME="+filepath.Join(outside, "xdg-data"),
	)

	version := exec.Command(copiedBinary, "version")
	version.Dir = outside
	version.Env = cmdEnv
	versionOut, err := version.CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, string(versionOut))
	}
	if strings.Contains(string(versionOut), " dev") {
		t.Fatalf("version output should not fall back to dev: %s", string(versionOut))
	}
	if !strings.Contains(string(versionOut), "namelens "+expectedVersion) {
		t.Fatalf("version output mismatch, expected %q in %q", "namelens "+expectedVersion, strings.TrimSpace(string(versionOut)))
	}

	envInfo := exec.Command(copiedBinary, "envinfo")
	envInfo.Dir = outside
	envInfo.Env = cmdEnv
	envInfoOut, err := envInfo.CombinedOutput()
	if err != nil {
		t.Fatalf("envinfo failed: %v\n%s", err, string(envInfoOut))
	}
	if strings.Contains(string(envInfoOut), "Config load failed") {
		t.Fatalf("envinfo indicates config loading failed outside repo: %s", string(envInfoOut))
	}
}
