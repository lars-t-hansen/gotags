# SPDX-License-Identifier: MIT

.PHONY: default

default:
	@echo "Pick an explicit target"

README.md: gotags Makefile
	./makedoc.sh

gotags: *.go utils/*.go
	go build

TAGS: gotags *.go utils/*.go
	./gotags *.go utils/*.go

