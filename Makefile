# SPDX-License-Identifier: MIT

.PHONY: default

default:
	@echo "Pick an explicit target"

README.md: gotags.go Makefile
	echo "# gotags - reliable etags-style tag information for Go" > README.md
	echo >> README.md
	go doc | expand -t4 | awk '/^func / { exit } { print }' >> README.md

TAGS: gotags *.go
	./gotags *.go
