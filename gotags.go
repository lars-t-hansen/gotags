// SPDX-License-Identifier: MIT

/*
Gotags generates an etags-like tag file for Go source, but with better Go awareness than etags.

Input file names are provided on the command line.  If the only input file name is given as "-" then
the names of input files are read from standard input, one name per line.

Usage:

	gotags [flags] input-filename ...

The flags are:

	-o output-filename
	    Write output to output-filename rather than to TAGS.  If output-filename
	    is "-" then write to standard output.

Tags are generated for all global names: packages, types, constants, functions, and variables,
irrespective of the declaration syntax.  (Etags does not handle constants or variables, nor types
defined inside type lists, nor functions or types with type parameters, and it can mistake local
type declarations for global ones.)

For full functionality, gotags requires each input file to be syntactically well-formed in the
sense of "go/parser".  If a file cannot be parsed, gotags prints a warning and falls back to
etags-style parsing.

Input file names are emitted verbatim in the output, there's no resolution of relative file names
wrt the location of the output file as in etags.  Nor is there support for other exotic etags
functionality, such as compressed files.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"iter"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	outname = flag.String("o", "TAGS", "`Filename` of output file, '-' for stdout")
)

// Output format.
//
// The full tag file syntax is described by etc/ETAGS.EBNF in the Emacs sources.  Gotags generates a
// file that does not use include sections, implicit tag names or file properties.  The simplified
// output grammar is:
//
//  tagfile    ::= tagsection*
//  tagsection ::= FF LF filename "," "0" tagdef* LF
//  filename   ::= filename-byte+
//  tagdef     ::= LF pattern DEL unsigned "," unsigned?
//  pattern    ::= pattern-byte+
//  unsigned   ::= [0-9]+
//  FF         ::= 0x0C
//  LF         ::= 0x0A
//  DEL        ::= 0x7F
//
// A pattern-byte is any byte value that does not include the three control characters.  It should
// encode a valid source character for Go.  It's unclear to me if Emacs does only 8-bit ASCII or can
// handle UTF8 here.
//
// A filename-byte is any byte value that is valid in a file name on the operating system in
// question, but not including "," or LF.
//
// In the tagdef, the unsigned values are zero-based line number and file offset for the start of
// the tag.  The original grammar allows for one or the other to be omitted.  gotags will never omit
// the line number, but may omit the file offset; Emacs seems to cope with that.
//
// The pattern includes the keyword that introduced it and all other text to the left of it from the
// start of the line: "func main" is usually the pattern for the main function, but should
// whitespace precede the "func" it is also included, as is additional whitespace between the two
// tokens.  In declaration lists for type, var, and const, the pattern is therefore normally the
// name and the whitespace preceding it.  And if there are multiple var or const names on a single
// line, the tags for the second and subsequent names will have the same offset and line number as
// the first, and their patterns will include all the preceding names too (being part of their
// literal left context).  Emacs seems to handle this fine.
//
// Note that etags would emit "func main(" where here we emit only "func main", but Emacs seems to
// cope with that.
//
// TODO: Ideally we should include right context in the pattern where that is sensible but we have
// to be careful.  The pattern must match the use.  For inferred type arguments, a call will not
// include the type argument list, so a generic function cannot be defined with the pattern "name[",
// since uses will be of the form "name(".  It's actually sort of interesting that a pattern like
// "func main" works, since "func" is not at the use site.

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: %s [options] filename ...\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var inputs iter.Seq[string]
	rest := flag.Args()
	if len(rest) == 1 && rest[0] == "-" {
		inputs = func(yield func(string) bool) {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if !yield(scanner.Text()) {
					break
				}
			}
		}
	} else {
		inputs = slices.Values(rest)
	}

	var output *os.File
	if *outname == "-" {
		output = os.Stdout
	} else {
		var err error
		output, err = os.Create(*outname)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
	}

	gotags(inputs, output, false)
}

var (
	fset = token.NewFileSet()
)

func gotags(inputs iter.Seq[string], output io.Writer, quiet bool) {
	for inputFn := range inputs {
		// Always emit the file header/footer even if the file defines no symbols (highly unusual,
		// as there's normally at least a package clause).
		fmt.Fprintf(output, "\x0C\x0A%s,0", inputFn)

		// We're going to need a handle on the source text no matter what.
		inputBytes, err := os.ReadFile(inputFn)
		if err != nil {
			if !quiet {
				log.Printf("Skipping %s: %v", inputFn, err)
			}
			continue
		}
		inputText := string(inputBytes)

		// Attempt to parse the file (ParseFile executes fset.AddFile).  If the parse succeeds,
		// generate info from the parse tree, otherwise fall back to a per-line regexp match a la
		// etags.
		f, err := parser.ParseFile(fset, inputFn, inputText, parser.SkipObjectResolution)
		if err == nil {
			semtags(inputFn, inputText, f, output)
		} else {
			if f != nil {
				if !quiet {
					log.Printf("Reverting to etags parsing for %s: %v", inputFn, err)
				}
				etags(inputText, output)
			} else {
				if !quiet {
					log.Printf("Skipping %s: %v", inputFn, err)
				}
			}
		}
		fmt.Fprintf(output, "\x0A")
	}
}

func semtags(inputFn, inputText string, f *ast.File, output io.Writer) {
	makeTag(inputText, f.Name, output)
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			makeTag(inputText, fd.Name, output)
			continue
		}
		if item, ok := d.(*ast.GenDecl); ok {
			if item.Tok != token.TYPE && item.Tok != token.VAR && item.Tok != token.CONST {
				continue
			}
			// It looks like Emacs will (a) find the ident under the point for M-. and then (b) find a pattern
			// that contains the ident and then (c) attempt to match the pattern against the text around the point,
			// but with the constraint that the pattern must match the text at the start of a line.  Therefore,
			// in lists we must extract the left context to the start of the line.  That is probably *also* true
			// in all other cases but there we don't see it yet.
			if item.Tok == token.TYPE {
				for _, spec := range item.Specs {
					makeTag(inputText, spec.(*ast.TypeSpec).Name, output)
				}
			} else {
				for _, spec := range item.Specs {
					vs := spec.(*ast.ValueSpec)
					for _, name := range vs.Names {
						makeTag(inputText, name, output)
					}
				}
			}
		}
	}
}

func makeTag(inputText string, name *ast.Ident, output io.Writer) {
	pos := name.NamePos
	tf := fset.File(pos)
	offs := tf.Offset(pos)
	line := tf.Line(pos) - 1
	end := offs + len(name.Name)
	for offs > 0 && inputText[offs-1] != '\n' {
		offs--
	}
	fmt.Fprintf(output, "\x0A%s\x7F%d,%d", inputText[offs:end], line, offs)
}

// This regexp is not actually etags-equivalent.  It requires the keyword to start in column 0,
// which is more limiting, but acceptable because that follows standard Go formatting for globals.
// On the positive side it also includes var/const definitions found in column 0, won't typically
// include types defined inside functions, and handles type parameters.  It still won't find
// var/const/type definitions inside lists, but that's why this is a fallback from the full parser.
// Like etags, it will be confused by code inside multi-line strings.

var etagsRe = regexp.MustCompile(`^(?:((?:package|func|type|var|const)\s+[a-zA-Z0-9_]+))`)

// Note we have no file offsets.

func etags(inputText string, output io.Writer) {
	scanner := bufio.NewScanner(strings.NewReader(inputText))
	lineno := 0
	for scanner.Scan() {
		l := scanner.Text()
		if m := etagsRe.FindStringSubmatch(l); m != nil {
			fmt.Fprintf(output, "\x0A%s\x7F%d,", m[1], lineno)
		}
		lineno++
	}
}
