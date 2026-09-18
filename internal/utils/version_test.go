package utils

import "testing"

// A6 — the comparison has to short circuit on the first differing field. It used
// to check every field independently, so a lower major could be rescued by a
// higher minor: {1,5,0} came out as newer than {2,0,0}.
func TestNewerThanComparesFieldByField(t *testing.T) {
	tests := []struct {
		name  string
		v     Version
		right Version
		want  bool
	}{
		{"major decides", Version{Major: 2}, Version{Major: 1, Minor: 9, Patch: 9}, true},
		{"a lower major is not rescued by a higher minor", Version{Major: 1, Minor: 5}, Version{Major: 2}, false},
		{"minor decides when the major matches", Version{Major: 1, Minor: 5}, Version{Major: 1, Minor: 4, Patch: 9}, true},
		{"a lower minor is not rescued by a higher patch", Version{Major: 1, Minor: 4, Patch: 9}, Version{Major: 1, Minor: 5}, false},
		{"patch decides", Version{Major: 1, Minor: 4, Patch: 2}, Version{Major: 1, Minor: 4, Patch: 1}, true},
		{"a release beats a prerelease of the same numbers", Version{Major: 1}, Version{Major: 1, Rc: true, PrereleaseID: 9}, true},
		{"a prerelease does not beat the release", Version{Major: 1, Beta: true, PrereleaseID: 9}, Version{Major: 1}, false},
		{"alpha < beta", Version{Major: 1, Beta: true}, Version{Major: 1, Alpha: true}, true},
		{"prerelease id decides inside one level", Version{Major: 1, Beta: true, PrereleaseID: 2}, Version{Major: 1, Beta: true, PrereleaseID: 1}, true},
		{"equal versions are not newer", Version{Major: 1, Minor: 2, Patch: 3}, Version{Major: 1, Minor: 2, Patch: 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.NewerThan(tt.right); got != tt.want {
				t.Fatalf("%v.NewerThan(%v) = %v, want %v", tt.v, tt.right, got, tt.want)
			}
		})
	}
}

// The same field by field comparison exists twice, and ShortVersion had the
// same bug.
func TestShortVersionNewerThanComparesFieldByField(t *testing.T) {
	if (ShortVersion{Major: 1, Minor: 5}).NewerThan(ShortVersion{Major: 2}) {
		t.Fatal("1.5 reported as newer than 2.0")
	}
	if !(ShortVersion{Major: 2}).NewerThan(ShortVersion{Major: 1, Minor: 9}) {
		t.Fatal("2.0 not reported as newer than 1.9")
	}
	if (ShortVersion{Major: 1, Minor: 2}).NewerThan(ShortVersion{Major: 1, Minor: 2}) {
		t.Fatal("equal short versions reported as newer")
	}
}

// A7 — the third capture group is always present in the match slice, it is just
// empty for a two part version. That is what made the old `len(matches) > 3`
// guard true and ParseInt("") panic.
func TestParseVersionAcceptsTwoPartVersions(t *testing.T) {
	matches := versionRegex.FindStringSubmatch("1.2")
	if len(matches) != 4 || matches[3] != "" {
		t.Fatalf("regexp result = %q, want a present but empty third group", matches)
	}

	ver, ok := ParseVersion("1.2")
	if !ok {
		t.Fatal(`ParseVersion("1.2") rejected a two part version`)
	}
	if ver.Major != 1 || ver.Minor != 2 || ver.Patch != 0 {
		t.Fatalf(`ParseVersion("1.2") = %v`, ver)
	}
}

// A7 — a number that does not fit must be reported, not panic: the callers parse
// yt-dlp output and GitHub tags, so the input is external.
func TestParseVersionReportsUnparsableNumbers(t *testing.T) {
	for _, text := range []string{"", "not a version", "99999999999.1", "1.99999999999"} {
		if _, ok := ParseVersion(text); ok {
			t.Fatalf("ParseVersion(%q) accepted nonsense", text)
		}
		if _, ok := ParseShortVersion(text); ok {
			t.Fatalf("ParseShortVersion(%q) accepted nonsense", text)
		}
	}
}
