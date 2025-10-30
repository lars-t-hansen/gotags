// SPDX-License-Identifier: MIT

// TODO: test cases
// - multiple input files
// - read file names from iterator (really from stdin, need to factor)
// - some file should trigger fallback to etags case

package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestSimple(t *testing.T) {

	// Each test file contains go code and each line that should give rise to a tag has a comment
	// that starts with //D followed by a list of expected tag patterns for that line (with literal
	// tabs if necessary) separated by |, eg "|var v1|var v1, v2|" for a var decl that introduces
	// two names.

	testFiles := []string{"testdata/t1.go"}

	var out strings.Builder
	gotags(slices.Values(testFiles), &out, true)
	outLines := strings.Split(out.String(), "\n")
	o := 0 // Line number

	for _, testFile := range testFiles {
		if len(outLines) < 2 || outLines[o] != "\x0C" || outLines[o+1] != fmt.Sprintf("%s,0", testFile) {
			t.Fatalf("%s: o=%d: Expected header in output", testFile, o)
		}
		o += 2

		inBytes, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("%s: Not readable: %v", testFile, err)
		}
		inLines := strings.Split(string(inBytes), "\n")
		i := 0  // Line number
		ix := 0 // Byte offset of line start

		for i < len(inLines) {
			if _, after, found := strings.Cut(inLines[i], "//D "); found {
				patterns := strings.Split(after, "|")
				if len(patterns) < 3 {
					t.Fatalf("%s: i=%d: Bad test case: %s", testFile, i, inLines[i])
				}
				patterns = patterns[1 : len(patterns)-1]
				for _, p := range patterns {
					if o == len(outLines) {
						t.Fatalf("%s: i=%d: Exhausted output on test case %s", testFile, i, inLines[i])
					}
					got := outLines[o]
					o++
					expect := fmt.Sprintf("%s\x7F%d,%d", p, i, ix)
					if got != expect {
						t.Fatalf("%s: i=%d: Got %s but expected %s\n", testFile, i, got, expect)
					}
				}
			}
			ix += len(inLines[i]) + 1
			i++
		}
		if o > len(outLines)-1 || outLines[o] != "" {
			t.Fatalf("%s: o=%d: Missing or bad footer", testFile, o)
		}
		o++
	}
	if o < len(outLines) {
		t.Fatalf("Excess output: o=%d %s", o, outLines[o])
	}
}
