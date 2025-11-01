// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	idAtEnd = regexp.MustCompile(`(` + identCharSet + `+)$`)
	commaAtEnd = regexp.MustCompile(`(,\s*)$`)
)

func TestTagging(t *testing.T) {

	// Each test file contains Go code and each line that should give rise to a tag has a comment
	// that starts with //D followed by a list of expected tag patterns for that line (with literal
	// tabs if necessary) separated by |, eg "|var v1|var v1, v2|" for a var decl that introduces
	// two names.  The tag names extracted from a pattern are the rightmost comma-separated
	// identifiers.
	//
	// The default assumption is that a file is well-formed Go, but it can switch modes by using a
	// "//etags" line.

	testFiles := []string{"testdata/t1.go", "testdata/t2.go"}

	var out strings.Builder
	gotags(slices.Values(testFiles), &out, true)
	outLines := strings.Split(out.String(), "\n")
	o := 0 // Line number

	for fileNo, testFile := range testFiles {
		isGo := true
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
			if strings.HasPrefix(inLines[i], "//etags") {
				isGo = false
			} else if _, after, found := strings.Cut(inLines[i], "//D "); found {
				patterns := strings.Split(after, "|")
				if len(patterns) < 3 {
					t.Fatalf("%s: i=%d: Bad test case: %s", testFile, i, inLines[i])
				}
				patterns = patterns[1 : len(patterns)-1]
				for _, pattern := range patterns {
					srch := pattern
					tagnames := make([]string, 0)
					for {
						m := idAtEnd.FindStringSubmatch(srch)
						if m == nil {
							t.Fatalf("%s: i=%d: Bad test case: %s", testFile, i, inLines[i])
						}
						tagnames = append(tagnames, m[1])
						srch = srch[:len(srch)-len(m[1:1])]
						m = commaAtEnd.FindStringSubmatch(srch)
						if m == nil {
							break
						}
						srch = srch[:len(srch)-len(m[1:1])]
					}
					for _, tagname := range tagnames {
						if o == len(outLines) {
							t.Fatalf("%s: i=%d: Exhausted output on test case %s", testFile, i, inLines[i])
						}
						got := outLines[o]
						o++
						var expect string
						if isGo {
							expect = fmt.Sprintf("%s\x7F%s\x01%d,%d", pattern, tagname, i, ix)
						} else {
							expect = fmt.Sprintf("%s\x7F%s\x01%d,", pattern, tagname, i)
						}
						if got != expect {
							t.Fatalf("%s: i=%d: Got %s but expected %s\n", testFile, i, got, expect)
						}
					}
				}
			}
			ix += len(inLines[i]) + 1
			i++
		}
		if o > len(outLines)-1 {
			t.Fatalf("%s: o=%d n=%d: Missing footer", testFile, o, len(outLines))
		}
		// The footer: if we're on the last file then the last line we see is an empty string
		// and we advance, but if there are more files we should see a line with FF, which will
		// be checked by the header check above.
		if fileNo == len(testFiles)-1 {
			if outLines[o] != "" {
				t.Fatalf("%s: Bad footer, want empty string, got %s", testFile, outLines[o])
			}
			o++
		}
	}
	if o < len(outLines) {
		t.Fatalf("Excess output: o=%d %s", o, outLines[o])
	}
}
