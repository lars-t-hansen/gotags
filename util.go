// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"io"
	"iter"
)

func generateLines(input io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				break
			}
		}
	}
}
