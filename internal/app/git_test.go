package app

import "testing"

func TestParseGitPorcelainBranchLine(t *testing.T) {
	tests := []struct {
		line       string
		wantUp     bool
		wantAhead  int
		wantBehind int
	}{
		{line: "## main", wantUp: false, wantAhead: 0, wantBehind: 0},
		{line: "## main...origin/main", wantUp: true, wantAhead: 0, wantBehind: 0},
		{line: "## main...origin/main [ahead 2]", wantUp: true, wantAhead: 2, wantBehind: 0},
		{line: "## main...origin/main [behind 3]", wantUp: true, wantAhead: 0, wantBehind: 3},
		{line: "## main...origin/main [ahead 1, behind 4]", wantUp: true, wantAhead: 1, wantBehind: 4},
	}

	for _, tc := range tests {
		gotUp, gotAhead, gotBehind := parseGitPorcelainBranchLine(tc.line)
		if gotUp != tc.wantUp || gotAhead != tc.wantAhead || gotBehind != tc.wantBehind {
			t.Fatalf("line %q: got up=%v ahead=%d behind=%d, want up=%v ahead=%d behind=%d",
				tc.line, gotUp, gotAhead, gotBehind, tc.wantUp, tc.wantAhead, tc.wantBehind)
		}
	}
}
