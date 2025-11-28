// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	idAtEnd        = regexp.MustCompile(`(` + identCharSet + `+)$`)
	commaAtEnd     = regexp.MustCompile(`(,\s*)$`)
	notInNameAtEnd = regexp.MustCompile(`([\t\f\r (),;=]*)$`)
)

// Each test file contains Go code and each line that should give rise to a tag has a comment
// that starts with //D followed by a list of expected tag patterns for that line (with literal
// tabs if necessary) separated by |, eg "|var v1|var v1, v2|" for a var decl that introduces
// two names.  The tag names extracted from a pattern are the rightmost comma-separated
// identifiers.
//
// The default assumption is that a file is well-formed Go, but it can switch modes by using a
// "//builtin-etags" line or a "//native-etags" line.

var testFiles = []string{"testdata/t1.go", "testdata/t2.go", "testdata/t3.c"}

const (
	mGotags = iota
	mBuiltinEtags
	mNativeEtags
)

func TestTagging(t *testing.T) {
	var out strings.Builder
	stdout = &out
	if r := runMain(append([]string{"-o", "-", "-q"}, testFiles...)); r != 0 {
		t.Fatalf("Exit %d", r)
	}
	outLines := strings.Split(out.String(), "\n")
	o := 0 // Line number

	for fileNo, testFile := range testFiles {
		var mode int = mGotags
		// Since we may run the system etags for some inputs, we can't count on the output byte size
		// being zero always.
		if len(outLines) < 2 || outLines[o] != "\x0C" || !strings.HasPrefix(outLines[o+1], testFile+",") {
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
			if strings.HasPrefix(inLines[i], "//builtin-etags") {
				mode = mBuiltinEtags
			} else if strings.HasPrefix(inLines[i], "//native-etags") {
				mode = mNativeEtags
			} else if _, after, found := strings.Cut(inLines[i], "//D "); found {
				patterns := strings.Split(after, "|")
				if len(patterns) < 3 {
					t.Fatalf("%s: i=%d: Bad test case: %s", testFile, i, inLines[i])
				}
				patterns = patterns[1 : len(patterns)-1]
				for _, pattern := range patterns {
					srch := pattern
					if m := notInNameAtEnd.FindStringSubmatch(srch); m != nil {
						srch = srch[:len(srch)-len(m[1])]
					}
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
						lineno := i + 1
						switch mode {
						case mGotags:
							expect = fmt.Sprintf("%s\x7F%s\x01%d,%d", pattern, tagname, lineno, ix)
						case mBuiltinEtags:
							expect = fmt.Sprintf("%s\x7F%s\x01%d,", pattern, tagname, lineno)
						case mNativeEtags:
							expect = fmt.Sprintf("%s\x7F%d,%d", pattern, lineno, ix)
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

// Filenames can be piped in via stdin, one per line
func TestPipedNames(t *testing.T) {
	outfile, err := os.CreateTemp("", "piped")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outfile.Name())
	stdin = strings.NewReader(strings.Join(testFiles, "\n"))
	var output strings.Builder
	stdout = &output
	stderr = &output
	if r := runMain([]string{"-", "-o", outfile.Name()}); r != 0 {
		t.Fatalf("Exit code %d", r)
	}
	// Normally, stderr will have some output b/c we're reverting to etags parsing
	scanner := bufio.NewScanner(outfile)
	filenames := maps.Collect(slices.All(testFiles))
	for scanner.Scan() {
		l := scanner.Text()
		for k, v := range filenames {
			if strings.HasPrefix(l, v+",") {
				delete(filenames, k)
				break
			}
		}
	}
	if len(filenames) != 0 {
		t.Fatalf("Names left behind: %v", filenames)
	}
}

// Fallback from full parser to naive built-in parser b/c not well-formed Go.
func TestFallback1(t *testing.T) {
	var o1, o2 strings.Builder
	stdout = &o1
	stderr = &o2
	if r := runMain([]string{"testdata/t2.go", "-v", "-o", "/dev/null"}); r != 0 {
		t.Fatalf("Exit code %d: %s", r, o2.String())
	}
	// Normally, stderr will have some output b/c we're reverting to etags parsing
	scanner := bufio.NewScanner(strings.NewReader(o1.String()))
	matched := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Builtin etags: testdata/t2.go") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("Did not see verbose output about fallback")
	}
}

// Fallback from full parser to external etags b/c not Go.
func TestFallback2(t *testing.T) {
	var o1, o2 strings.Builder
	stdout = &o1
	stderr = &o2
	if r := runMain([]string{"testdata/t3.c", "-v", "-o", "/dev/null"}); r != 0 {
		t.Fatalf("Exit code %d: %s", r, o2.String())
	}
	// Normally, stderr will have some output b/c we're reverting to etags parsing
	scanner := bufio.NewScanner(strings.NewReader(o1.String()))
	matched := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "System etags: testdata/t3.c") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("Did not see verbose output about fallback")
	}
}
