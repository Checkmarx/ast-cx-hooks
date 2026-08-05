package cursor

import "testing"

// Cursor reports a Windows workspace root with a leading slash before the drive
// letter ("/c:/foo/bar"). Left unnormalized, that spelling flows into ast-cli's
// --ignored-file-path/--data @<file> arguments and its ignore-file reader, where
// Go's os.Open/os.ReadFile reject it outright. See normalizeWorkDir's doc comment
// in adapters.go for the full failure chain.
func TestNormalizeWorkDir(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"posix-style windows root", "/c:/Cx-Flow/Test/JavaVulnerabilityLabE", "c:/Cx-Flow/Test/JavaVulnerabilityLabE"},
		{"posix-style windows root, uppercase drive", "/C:/Users/dev/project", "C:/Users/dev/project"},
		{"native windows backslash root", `c:\Users\dev\project`, "c:/Users/dev/project"},
		{"native windows forward-slash root", "c:/Users/dev/project", "c:/Users/dev/project"},
		{"posix root, no drive letter", "/home/dev/project", "/home/dev/project"},
		{"empty", "", ""},
		{"drive letter only, no path", "/c:", "c:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeWorkDir(tc.in); got != tc.want {
				t.Errorf("normalizeWorkDir(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
