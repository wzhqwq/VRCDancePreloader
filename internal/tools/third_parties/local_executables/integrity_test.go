package local_executables

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// stubIcacls replaces the icacls runner and returns a function reporting the
// arguments of every call so far.
//
// The real call reads (and, for a Low level, rewrites) the ACLs of a file on the
// machine running the test, so this seam is what makes "was the executable
// inspected, and was it changed?" observable at all.
func stubIcacls(t *testing.T, output string) func() [][]string {
	t.Helper()

	var mu sync.Mutex
	var args [][]string

	previous := runIcacls
	runIcacls = func(ctx context.Context, call ...string) ([]byte, error) {
		mu.Lock()
		args = append(args, append([]string(nil), call...))
		mu.Unlock()

		return []byte(output), nil
	}

	t.Cleanup(func() { runIcacls = previous })

	return func() [][]string {
		mu.Lock()
		defer mu.Unlock()

		return append([][]string(nil), args...)
	}
}

// icaclsOutputWithoutLabel is what icacls prints for an ordinary file: no
// "Mandatory Label" line at all, because nobody ever set an integrity level on
// it, and such a file simply runs at the default Medium.
const icaclsOutputWithoutLabel = `C:\binaries\tool.exe BUILTIN\Administrators:(I)(F)
                NT AUTHORITY\SYSTEM:(I)(F)

Successfully processed 1 files; Failed processing 0 files
`

// icaclsOutputWithLabel is what icacls prints once an integrity level has been
// set on the file. "Mandatory Label" is the label's *name* in that output, not
// translatable text.
func icaclsOutputWithLabel(level string) string {
	return `C:\binaries\tool.exe Mandatory Label\` + level + ` Mandatory Level:(NW)`
}

// installTestBinary puts a file that LookPath accepts where the test wants it.
//
// Only the extension matters: the child itself never starts, because everything
// these tests assert happens before that.
func installTestBinary(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create the binaries directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("not a real executable"), 0o755); err != nil {
		t.Fatalf("install the test binary at %s: %v", path, err)
	}
}

// C1-② — the integrity level is inspected before the executable is started, and
// no longer while SetPathAndCheck holds the write lock.
//
// A Low integrity executable is refused by Windows the moment it starts, so the
// check is what makes yt-dlp work at all — but it runs up to two icacls processes
// with a 3s timeout each, and SetPathAndCheck holds the write lock that the GUI
// and the video request path wait on. The check therefore moved to the exec path,
// where it is guaranteed before the child runs and paid once per installed file.
func TestExecuteInspectsTheIntegrityLevelBeforeStarting(t *testing.T) {
	root := isolateAppData(t)

	binaryPath := filepath.Join(root, "binaries", "vrcdp-integrity-test.exe")
	installTestBinary(t, binaryPath)

	calls := stubIcacls(t, icaclsOutputWithoutLabel)

	d := NewDownloadableBinary("ytdlp")
	d.SetPathAndCheck(binaryPath)

	if got := len(calls()); got != 0 {
		t.Fatalf("inspections = %d after SetPathAndCheck, want 0: the check must not run under its write lock", got)
	}

	// The child fails to start, which is fine: the inspection has to happen before
	// the attempt either way.
	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := len(calls()); got != 1 {
		t.Fatalf("inspections = %d after Execute, want the executable inspected exactly once before it was started", got)
	}

	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := len(calls()); got != 1 {
		t.Fatalf("inspections = %d after a second Execute, want the already inspected file to be left alone", got)
	}

	// A different path is a different file, and nothing has inspected that one.
	secondPath := filepath.Join(root, "binaries", "vrcdp-integrity-test-2.exe")
	installTestBinary(t, secondPath)

	d.SetPathAndCheck(secondPath)

	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := len(calls()); got != 2 {
		t.Fatalf("inspections = %d after the path changed, want the new file inspected once as well", got)
	}
}

// C2-①（修正后的判定）—— 只有真的带 Low 标签的文件才被提升。
//
// "Mandatory Label" 是系统只给**显式设置过**完整性级别的文件加上的标签名，不是会被
// 本地化的输出文本：没有这一行的文件是**常态**（默认按 Medium 运行，什么都不用做），
// 所以正则不匹配既不是失败也不值得警告。整个检查因此只挂在"这一行存在且是 Low"上，
// 其余情况都必须原样离开文件。
func TestCheckIntegrityLevelOnlyRaisesLowLabels(t *testing.T) {
	root := isolateAppData(t)

	cases := []struct {
		name       string
		output     string
		wantRaised bool
	}{
		{name: "no label at all (the ordinary file)", output: icaclsOutputWithoutLabel, wantRaised: false},
		{name: "explicit Medium label", output: icaclsOutputWithLabel("Medium"), wantRaised: false},
		{name: "explicit High label", output: icaclsOutputWithLabel("High"), wantRaised: false},
		{name: "explicit Low label", output: icaclsOutputWithLabel("Low"), wantRaised: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binaryPath := filepath.Join(root, "binaries", "vrcdp-level-test.exe")
			installTestBinary(t, binaryPath)

			calls := stubIcacls(t, tc.output)

			d := NewDownloadableBinary("ytdlp")
			d.SetPathAndCheck(binaryPath)

			if _, err := d.Execute(context.Background(), "--version"); err == nil {
				t.Fatal("Execute succeeded on a file that is not an executable")
			}

			recorded := calls()

			// One inspection, plus the raise when the label asks for it.
			want := 1
			if tc.wantRaised {
				want = 2
			}
			if len(recorded) != want {
				t.Fatalf("icacls calls = %d, want %d: %v", len(recorded), want, recorded)
			}

			// The first call is always the plain inspection.
			if len(recorded[0]) != 1 {
				t.Fatalf("first icacls call = %v, want a plain inspection of the executable", recorded[0])
			}

			raised := false
			for _, call := range recorded {
				if len(call) > 1 && call[1] == "/setintegritylevel" {
					raised = true
				}
			}

			if raised != tc.wantRaised {
				t.Fatalf("raised the level = %v for the output %q, want %v", raised, tc.output, tc.wantRaised)
			}
		})
	}
}
