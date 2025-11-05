// SPDX-License-Identifier: MIT
// From https://github.com/lars-t-hansen/util/go-utils

package utils

import (
	"bufio"
	"io"
	"iter"
)

func GenerateLinesFromReader(input io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				break
			}
		}
	}
}
