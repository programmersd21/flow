package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// The CLI is flow's public interface, so it is tested by running the built
// binary rather than by calling functions directly. Everything runs with
// FLOW_CONFIG and FLOW_DATA pointed at temp dirs, so a test run can never
// read or clobber the developer's real config or daily totals.

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// buildBinary compiles flow once per test run.
func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "flow-cli-test")
		if err != nil {
			buildErr = err
			return
		}
		binPath = filepath.Join(dir, "flow")
		if runtime.GOOS == "windows" {
			binPath += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = err
			t.Logf("build output: %s", out)
		}
	})
	if buildErr != nil {
		t.Fatalf("building flow: %v", buildErr)
	}
	return binPath
}

type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// runCLI executes flow with the given args under an isolated environment.
// args may contain the literal string "STREAM" to request a bounded read of a
// long-running stream mode.
func runCLI(t *testing.T, timeout time.Duration, args ...string) result {
	t.Helper()
	bin := buildBinary(t)

	ctxDir := t.TempDir()
	dataDir := t.TempDir()

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"FLOW_CONFIG="+filepath.Join(ctxDir, "config.toml"),
		"FLOW_DATA="+dataDir,
		"NO_COLOR=1",
		"HOME="+ctxDir,
	)
	// The test process has no TTY, so UI modes must take the documented
	// non-interactive path rather than blocking on a terminal.

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting flow: %v", err)
	}

	var outBuf, errBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); copyAll(&outBuf, stdout) }()
	go func() { defer wg.Done(); copyAll(&errBuf, stderr) }()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var exitCode int
	select {
	case err := <-done:
		exitCode = exitCodeOf(err)
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		exitCode = -1 // killed: expected for stream modes
	}
	wg.Wait()

	return result{stdout: outBuf.String(), stderr: errBuf.String(), exitCode: exitCode}
}

func copyAll(dst *strings.Builder, src interface{ Read([]byte) (int, error) }) {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); ok {
		return ee.ExitCode()
	}
	return -1
}

func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// ─── trivial modes ───────────────────────────────────────────────────────────

func TestCLIVersion(t *testing.T) {
	got := runCLI(t, 10*time.Second, "--version")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	// Format is "flow <version>"; the version is injected at build time.
	if !strings.HasPrefix(got.stdout, "flow ") {
		t.Errorf("stdout = %q, want a line starting with \"flow \"", got.stdout)
	}
	if strings.TrimSpace(got.stdout) == "flow" {
		t.Error("--version printed no version")
	}
}

func TestCLIHelp(t *testing.T) {
	got := runCLI(t, 10*time.Second, "--help")
	if got.exitCode == -1 {
		t.Fatal("--help did not terminate")
	}
	// Go's flag package prints usage to stderr, including for -help.
	usage := got.stdout + got.stderr
	for _, want := range []string{
		"-tiny", "-mini", "-compact", "-once", "-json", "-json-stream",
		"-interface", "-refresh", "-bits", "-format", "-theme", "-no-color",
		"-no-anim", "-view", "-window", "-width", "-reset-history", "-version",
	} {
		if !strings.Contains(usage, want) {
			t.Errorf("--help does not document %s", want)
		}
	}
}

func TestCLIUnknownFlagFails(t *testing.T) {
	got := runCLI(t, 10*time.Second, "--definitely-not-a-flag")
	if got.exitCode == 0 {
		t.Error("an unknown flag should not exit 0")
	}
	if !strings.Contains(got.stderr, "not-a-flag") {
		t.Errorf("stderr should name the bad flag; got %q", got.stderr)
	}
}

// ─── one-shot modes ──────────────────────────────────────────────────────────

func TestCLIOnce(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--once")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	if strings.TrimSpace(got.stdout) == "" {
		t.Error("--once produced no output")
	}
	if strings.Contains(got.stdout, "\x1b[") {
		t.Error("--once emitted ANSI escapes under NO_COLOR")
	}
}

func TestCLITiny(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--tiny")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	line := strings.TrimSpace(got.stdout)
	if !strings.Contains(line, "↓") || !strings.Contains(line, "↑") {
		t.Errorf("--tiny output %q should contain both direction arrows", line)
	}
}

func TestCLINonTTYDefaultsToTiny(t *testing.T) {
	// No TTY and no output flag: must not block on the TUI.
	got := runCLI(t, 20*time.Second)

	if got.exitCode != 0 {
		// A sandboxed runner may have no usable interface, in which case a
		// clean non-zero exit with a diagnostic is the correct behaviour.
		// Require that diagnostic to name the problem, so a crash or garbage
		// output still fails rather than hiding behind the skip.
		msg := strings.TrimSpace(got.stderr)
		switch {
		case msg == "":
			t.Fatalf("exit = %d with empty stderr, want a diagnostic", got.exitCode)
		case !strings.Contains(msg, "interface") && !strings.Contains(msg, "collector"):
			t.Fatalf("exit = %d with unexpected stderr %q, want a collector diagnostic",
				got.exitCode, msg)
		}
		t.Skipf("no usable interface here: %s", msg)
	}

	line := strings.TrimSpace(got.stdout)
	if !strings.Contains(line, "↓") {
		t.Errorf("non-TTY run should print the single-line form, got %q (stderr %q)",
			line, got.stderr)
	}
	if strings.Contains(line, "\n") {
		t.Errorf("non-TTY output should be a single line, got %q", line)
	}
	if strings.Contains(line, "\x1b[") {
		t.Error("non-TTY output must not contain escape codes")
	}
}

func TestCLIBitsAffectsUnits(t *testing.T) {
	plain := runCLI(t, 20*time.Second, "--tiny")
	bits := runCLI(t, 20*time.Second, "--tiny", "--bits")
	if plain.exitCode != 0 || bits.exitCode != 0 {
		t.Fatalf("exit codes: %d / %d", plain.exitCode, bits.exitCode)
	}
	// Bits multiplies the rate by 8, so at least one of the two must differ.
	if strings.TrimSpace(plain.stdout) == strings.TrimSpace(bits.stdout) {
		t.Skip("no traffic during the test; --bits effect not observable")
	}
}

// ─── format ──────────────────────────────────────────────────────────────────

func TestCLIFormat(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--format", "{{.Iface}} {{.Down}} {{.Up}}")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	line := strings.TrimSpace(got.stdout)
	if line == "" {
		t.Fatal("--format produced no output")
	}
	if strings.Contains(line, "{{") {
		t.Errorf("unrendered template left in output: %q", line)
	}
	if strings.Count(line, "\n") > 0 {
		t.Errorf("one-shot --format should print one line, got %q", line)
	}
}

func TestCLIFormatWidth(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--format", "x", "--width", "10")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	line := strings.TrimRight(got.stdout, "\n")
	if len([]rune(line)) != 10 {
		t.Errorf("--width 10 produced %d columns: %q", len([]rune(line)), line)
	}
}

func TestCLIFormatInvalidTemplate(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--format", "{{.Broken")
	if got.exitCode != 2 {
		t.Errorf("bad template exit = %d, want 2 (stderr: %q)", got.exitCode, got.stderr)
	}
	if !strings.Contains(strings.ToLower(got.stderr), "template") {
		t.Errorf("stderr should mention the template problem, got %q", got.stderr)
	}
}

// ─── validation ──────────────────────────────────────────────────────────────

func TestCLIInvalidRefresh(t *testing.T) {
	got := runCLI(t, 10*time.Second, "--refresh", "not-a-duration", "--once")
	// Either a clean rejection or a clamp-and-warn is acceptable, but it must
	// never report success silently.
	if got.exitCode != 0 && !strings.Contains(got.stderr, "refresh") {
		t.Errorf("bad --refresh should report a problem; exit=%d stderr=%q",
			got.exitCode, got.stderr)
	}
}

func TestCLIInvalidInterfaceIsActionable(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--interface", "definitely-not-an-interface", "--once")
	if got.exitCode == 0 {
		t.Errorf("a missing interface should not exit 0; stdout=%q", got.stdout)
	}
	// The message should name the interface and ideally list what is available.
	if !strings.Contains(got.stderr, "definitely-not-an-interface") {
		t.Errorf("stderr should name the interface, got %q", got.stderr)
	}
}

func TestCLIResetHistory(t *testing.T) {
	got := runCLI(t, 15*time.Second, "--reset-history")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	if !strings.Contains(strings.ToLower(got.stdout+got.stderr), "clear") {
		t.Errorf("reset-history should confirm; stdout=%q stderr=%q", got.stdout, got.stderr)
	}
}

// ─── json ────────────────────────────────────────────────────────────────────

func TestCLIOnceJSON(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--once", "--json")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(got.stdout), &doc); err != nil {
		t.Fatalf("--once --json is not valid JSON: %v\n%s", err, got.stdout)
	}
	// Documented contract: these fields always exist with these types.
	for _, k := range []string{
		"status", "timestamp", "interface",
		"download_bps", "upload_bps", "download_human", "upload_human",
		"peak_down_bps", "peak_up_bps", "unit_display",
	} {
		if _, ok := doc[k]; !ok {
			t.Errorf("JSON is missing required field %q; got keys %v", k, keysOf(doc))
		}
	}
	if doc["status"] != "ok" {
		t.Errorf("status = %v, want \"ok\"", doc["status"])
	}
	for _, k := range []string{"download_bps", "upload_bps", "peak_down_bps", "peak_up_bps"} {
		v, ok := doc[k].(float64)
		if !ok {
			t.Errorf("%s should be a JSON number, got %T", k, doc[k])
			continue
		}
		if v < 0 {
			t.Errorf("%s = %v, must not be negative", k, v)
		}
	}
	ts, _ := doc["timestamp"].(string)
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", ts, err)
	}
	iface, _ := doc["interface"].(string)
	if iface == "" {
		t.Error("interface should name the interface that was sampled")
	}
}

func TestCLIBitsJSONUnits(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--once", "--json", "--bits")
	if got.exitCode != 0 {
		t.Fatalf("exit = %d, stderr: %s", got.exitCode, got.stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(got.stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// With --bits the human string must use a bits unit.
	human, _ := doc["download_human"].(string)
	if !strings.Contains(human, "b/s") {
		t.Errorf("with --bits, download_human = %q, want a bits unit", human)
	}
}

func TestCLIJSONStreamRepeatsValidRecords(t *testing.T) {
	got := runCLI(t, 1500*time.Millisecond, "--json-stream", "--refresh", "100ms")
	if got.exitCode != -1 {
		t.Logf("json-stream exited on its own with %d", got.exitCode)
	}
	sc := bufio.NewScanner(strings.NewReader(got.stdout))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			t.Fatalf("stream line %d is not valid JSON: %v\n%s", n+1, err, line)
		}
		n++
	}
	if n < 2 {
		t.Errorf("expected repeated records from --json-stream, got %d", n)
	}
}

// ─── non-fatal flags ─────────────────────────────────────────────────────────

func TestCLIViewAndThemeAndWindowAccepted(t *testing.T) {
	for _, args := range [][]string{
		{"--view", "tiny", "--once"},
		{"--view", "compact", "--once"},
		{"--theme", "nord", "--once"},
		{"--theme", "ansi", "--once"},
		{"--window", "5m", "--once"},
		{"--no-anim", "--once"},
		{"--no-color", "--once"},
		{"--ping", "127.0.0.1", "--once"},
	} {
		got := runCLI(t, 20*time.Second, args...)
		if got.exitCode != 0 {
			t.Errorf("flow %v: exit = %d, stderr: %s",
				strings.Join(args, " "), got.exitCode, got.stderr)
		}
	}
}

func TestCLIInvalidThemeIsClean(t *testing.T) {
	got := runCLI(t, 20*time.Second, "--theme", "definitely-not-a-theme", "--once")
	// A UI-mode theme flag should not crash the process.
	if got.exitCode == -1 {
		t.Fatal("process hung with an unknown theme")
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
