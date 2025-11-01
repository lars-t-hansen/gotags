// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"testing"
)

func TestGenerateLines(t *testing.T) {
	f, err := os.Open("testdata/lines.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n := 0
	for l := range generateLines(f) {
		expect := fmt.Sprint(n)
		n++
		if l != expect {
			t.Fatalf("Got <%s> Expected <%s>", l, expect)
		}
	}
	if n != 11 {
		t.Fatalf("Expected 11 lines, n=%d", n)
	}
}
