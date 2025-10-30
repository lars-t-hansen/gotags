package main

import (
	"fmt"
	"os"
	"strings"
	"slices"
	"testing"
)

func TestSimple(t *testing.T) {
	var out strings.Builder
	gotags(slices.Values([]string{"testdata/t1.go"}), &out)
	outLines := strings.Split(out.String(), "\n")

	if outLines[0] != "\x0C" || outLines[1] != "testdata/t1.go,0" {
		t.Fatalf("Bad header %s %s\n", outLines[0], outLines[1])
	}
	o := 2

	inBytes, err := os.ReadFile("testdata/t1.go")
	if err != nil {
		t.Fatal(err)
	}
	inLines := strings.Split(string(inBytes), "\n")

	i := 0						// Line number
	ix := 0						// Byte offset of line start
	for {
		if _, after, found := strings.Cut(inLines[i], "//D "); found {
			patterns := strings.Split(after, "|")
			if len(patterns) < 3 {
				t.Fatalf("Bad test case: %s", inLines[i])
			}
			patterns = patterns[1:len(patterns)-1]
			for _, p := range patterns {
				if o == len(outLines) {
					t.Fatalf("Exhausted output on test case %s", inLines[i])
				}
				got := outLines[o]
				o++
				expect := fmt.Sprintf("%s\x7F%d,%d", p, i, ix)
				if got != expect {
					t.Fatalf("Failed: got %s expected %s\n", got, expect)
				} else {
					t.Logf("OK %s %s", got, expect)
				}
			}
		}
		ix += len(inLines[i]) + 1
		i++
	}
}
