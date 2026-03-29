package cli

import (
	"bytes"
	"strings"
	"testing"
)

// helper runs Run() with --dir pointing to an isolated temp directory and
// returns stdout, stderr contents plus the exit code.
func runCLI(t *testing.T, args []string) (stdout, stderr string, code int) {
	t.Helper()
	dir := t.TempDir()
	fullArgs := append([]string{"--dir", dir}, args...)
	var out, errBuf bytes.Buffer
	code = Run(fullArgs, &out, &errBuf)
	return out.String(), errBuf.String(), code
}

// ---------------------------------------------------------------------------
// 1. Dispatch: help, version, unknown
// ---------------------------------------------------------------------------

func TestRunHelp(t *testing.T) {
	stdout, _, code := runCLI(t, []string{"help"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "bujotui") {
		t.Errorf("help output should mention bujotui, got: %s", stdout)
	}
}

func TestRunVersion(t *testing.T) {
	stdout, _, code := runCLI(t, []string{"version"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "bujotui") {
		t.Errorf("version output should contain 'bujotui', got: %s", stdout)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	_, stderr, code := runCLI(t, []string{"nosuchcmd"})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr should mention unknown command, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 2. cmdAdd via Run()
// ---------------------------------------------------------------------------

func TestAddBasic(t *testing.T) {
	stdout, stderr, code := runCLI(t, []string{"add", "Buy milk"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Buy milk") {
		t.Errorf("stdout should contain description, got: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// 3. cmdAdd with flags
// ---------------------------------------------------------------------------

func TestAddWithFlags(t *testing.T) {
	stdout, stderr, code := runCLI(t, []string{"add", "-s", "event", "-p", "work", "Meeting"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "event") {
		t.Errorf("stdout should contain symbol name 'event', got: %s", stdout)
	}
	if !strings.Contains(stdout, "work") {
		t.Errorf("stdout should contain project 'work', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Meeting") {
		t.Errorf("stdout should contain description 'Meeting', got: %s", stdout)
	}
}

// ---------------------------------------------------------------------------
// 4. cmdAdd empty description
// ---------------------------------------------------------------------------

func TestAddEmptyDescription(t *testing.T) {
	_, stderr, code := runCLI(t, []string{"add"})
	if code != 1 {
		t.Fatalf("expected exit 1 for empty description, got %d", code)
	}
	if !strings.Contains(stderr, "description required") {
		t.Errorf("stderr should mention description required, got: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 5. cmdList via Run(): add entries then list
// ---------------------------------------------------------------------------

func TestListAfterAdd(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer

	// Add an entry
	code := Run([]string{"--dir", dir, "add", "Test entry for list"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("add failed: exit %d; stderr: %s", code, errBuf.String())
	}

	// List entries (today)
	out.Reset()
	errBuf.Reset()
	code = Run([]string{"--dir", dir, "list"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("list failed: exit %d; stderr: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Test entry for list") {
		t.Errorf("list output should contain the added entry, got: %s", out.String())
	}
}

// ---------------------------------------------------------------------------
// 6. cmdList --week: verify no error
// ---------------------------------------------------------------------------

func TestListWeek(t *testing.T) {
	_, stderr, code := runCLI(t, []string{"list", "--week"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
}

// ---------------------------------------------------------------------------
// 7. cmdModify (done): add entry then mark done
// ---------------------------------------------------------------------------

func TestDoneModify(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer

	// Add a task
	code := Run([]string{"--dir", dir, "add", "Finish report"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("add failed: exit %d; stderr: %s", code, errBuf.String())
	}

	// Mark entry 1 as done
	out.Reset()
	errBuf.Reset()
	code = Run([]string{"--dir", dir, "done", "1"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("done failed: exit %d; stderr: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "done") {
		t.Errorf("done output should contain 'done', got: %s", out.String())
	}
	if !strings.Contains(out.String(), "Finish report") {
		t.Errorf("done output should contain entry description, got: %s", out.String())
	}
}

// ---------------------------------------------------------------------------
// 8. cmdRemove: add entry then remove
// ---------------------------------------------------------------------------

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer

	// Add an entry
	code := Run([]string{"--dir", dir, "add", "Delete me"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("add failed: exit %d; stderr: %s", code, errBuf.String())
	}

	// Remove entry 1
	out.Reset()
	errBuf.Reset()
	code = Run([]string{"--dir", dir, "remove", "1"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("remove failed: exit %d; stderr: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("remove output should contain 'removed', got: %s", out.String())
	}
	if !strings.Contains(out.String(), "Delete me") {
		t.Errorf("remove output should contain entry description, got: %s", out.String())
	}

	// Verify entry is gone by listing
	out.Reset()
	errBuf.Reset()
	code = Run([]string{"--dir", dir, "list"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("list failed: exit %d; stderr: %s", code, errBuf.String())
	}
	if strings.Contains(out.String(), "Delete me") {
		t.Errorf("list should not contain removed entry, got: %s", out.String())
	}
}

// ---------------------------------------------------------------------------
// 9. Config subcommand: returns 0
// ---------------------------------------------------------------------------

func TestConfig(t *testing.T) {
	stdout, stderr, code := runCLI(t, []string{"config"})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Config dir") {
		t.Errorf("config output should contain 'Config dir', got: %s", stdout)
	}
}
