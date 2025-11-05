// SPDX-License-Identifier: MIT

/*
Gotags generates an etags-like tag file for Go source, with better Go awareness than etags.

Input file names are provided on the command line.  If the only input file name is given as "-" then
the names of input files are read from standard input, one name per line.

Input files with extension other than .go are processed by the native etags into the specified output
file.

Usage:

	gotags [flags] input-filename ...

The flags are:

	-o output-filename
	    Write output to output-filename rather than to TAGS.  If output-filename
	    is "-" then write to standard output.

	--etags pathname
		The name of the native etags command if not /usr/bin/etags, or specify
		the empty string to disable the use of native etags for non-Go files.

	-V, --version
		Print version information and exit.

	-h
		Print help and exit.

Tags are generated for all Go global names: packages, types, constants, functions, variables, and
members of global interfaces, irrespective of the declaration syntax.  In contrast, etags does not
handle constants or variables, nor types defined inside type lists, nor functions or types with
type parameters, nor interface members, and it can mistake local type declarations for global ones.

For full functionality, gotags requires each Go input file to be syntactically well-formed in the
sense of "go/parser".  If a .go file cannot be parsed, gotags prints a warning and falls back to
its own etags-style parsing.

Input file names are emitted verbatim in the output, gotags has no resolution of relative file names
wrt the location of the output file as in etags, nor has it support for other exotic etags
functionality, such as compressed files.

Files that are passed to the native etags are processed entirely according to etags's semantics.

To use gotags with Emacs's etags-regen-mode or complete-symbol it is sufficient to set
etags-program-name to "gotags" in your .emacs.  Note however that gotags does not yet respect any
regular expression settings in that mode for any language.
*/
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"iter"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"

	"gotags/utils"
)

const VERSION = "0.3.0-devel"

// Command line arguments
var (
	outname            = "TAGS"
	systemEtagsCommand = "/usr/bin/etags"
	verbose            = false
	inputFilenames     = make([]string, 0)
)

func main() {
	parseArguments()

	var inputs iter.Seq[string]
	if len(inputFilenames) == 1 && inputFilenames[0] == "-" {
		inputs = utils.GenerateLinesFromReader(os.Stdin)
	} else {
		inputs = slices.Values(inputFilenames)
	}

	var output *os.File
	if outname == "-" {
		output = os.Stdout
	} else {
		var err error
		output, err = os.Create(outname)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
	}

	computeTags(inputs, output, false)
}

// Annoyingly, Emacs will invoke us as `gotags - -o fn` which the Go parser does not handle
// directly.  So we implement our own parsing.
//
// etags prints help and version on stdout, so we do too.

func parseArguments() {
	n := len(os.Args)
	i := 1
	bad := false
	for i < n && !bad {
		arg := os.Args[i]
		i++
		switch arg {
		case "-h":
			fmt.Printf(
				`Usage: gotags [options] input-filename ...

Input-filename can be "-" to denote that filenames will be read from stdin.

Options:

--etags pathname
  Path of the native etags program, "" to disable this functionality, default "%s"
-o filename
  Filename of output file, "-" for stdout, default "%s"
-v
  Enable verbose output (for debugging).
-V, --version
  Print version information.
`,
				systemEtagsCommand, outname)
			os.Exit(0)

		case "-o":
			if i == n {
				bad = true
			} else {
				outname = os.Args[i]
				i++
			}

		case "-v":
			verbose = true

		case "-V", "--version":
			fmt.Printf("gotags v%s (etags compatible)\n", VERSION)
			os.Exit(0)

		case "--etags":
			if i == n {
				bad = true
			} else {
				systemEtagsCommand = os.Args[i]
				i++
			}

		default:
			if arg[0] == '-' && len(arg) > 1 {
				bad = true
			} else {
				inputFilenames = append(inputFilenames, arg)
			}
		}
	}
	if bad {
		fmt.Fprintf(os.Stderr, "Bad command line arguments.  Try -h.")
		os.Exit(2)
	}
}

var fset = token.NewFileSet()

func computeTags(inputs iter.Seq[string], output io.Writer, quiet bool) {
	unhandledFiles := make([]string, 0)
	for inputFn := range inputs {
		if path.Ext(inputFn) != ".go" {
			unhandledFiles = append(unhandledFiles, inputFn)
			continue
		}
		fmt.Fprintf(output, "\x0C\x0A%s,0", inputFn)

		inputBytes, err := os.ReadFile(inputFn)
		if err != nil {
			if !quiet {
				log.Printf("Skipping %s: %v", inputFn, err)
			}
			continue
		}
		inputText := string(inputBytes)

		f, err := parser.ParseFile(fset, inputFn, inputText, parser.SkipObjectResolution)
		if err == nil {
			goTags(inputFn, inputText, f, output)
		} else {
			if !quiet {
				log.Printf("Reverting to etags parsing for %s: %v", inputFn, err)
			}
			builtinEtags(inputFn, inputText, output)
		}

		fmt.Fprintf(output, "\x0A")
	}
	if len(unhandledFiles) > 0 && systemEtagsCommand != "" {
		systemEtags(unhandledFiles, output)
	}
}

// Format for goTags-generated and builtinEtags-generated output.
//
// The full tag file syntax and a fair bit of its semantics are described by etc/ETAGS.EBNF in the
// Emacs sources.  Gotags generates a file that does not use "include" sections or file properties,
// always has explicit tag names, always has "0" for the size of the tagsection, and always emits
// line numbers.  The simplified output grammar is:
//
//  tagfile    ::= tagsection*
//  tagsection ::= FF LF filename "," "0" tagdef* LF
//  filename   ::= filename-byte+
//  tagdef     ::= LF pattern DEL tagname SOH lineno "," offset?
//  pattern    ::= pattern-byte+
//  tagname    ::= ident-char+
//  lineno     ::= unsigned, one-based
//  offset     ::= unsigned, zero-based
//  unsigned   ::= [0-9]+
//  SOH        ::= 0x01
//  FF         ::= 0x0C
//  LF         ::= 0x0A
//  DEL        ::= 0x7F
//
// A pattern-byte is any byte value that does not include the three control characters.  It should
// encode a valid source character for Go.  It's unclear to me if Emacs does only 8-bit ASCII or can
// handle UTF8 here.
//
// An ident-byte is any byte that can be part of a Go identifier.
//
// A filename-byte is any byte value that is valid in a file name on the operating system in
// question, but not including "," or LF.
//
// Per the standard semantics, as we do not use implicit tags the pattern always ends with the
// tagname.

func goTags(inputFn, inputText string, f *ast.File, output io.Writer) {
	if verbose {
		log.Printf("Gotags: %s", inputFn)
	}
	makeTag(inputText, f.Name, output)
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			makeTag(inputText, fd.Name, output)
			continue
		}
		if item, ok := d.(*ast.GenDecl); ok {
			switch item.Tok {
			case token.TYPE:
				for _, spec := range item.Specs {
					ts := spec.(*ast.TypeSpec)
					makeTag(inputText, ts.Name, output)
					if it, ok := ts.Type.(*ast.InterfaceType); ok {
						for _, field := range it.Methods.List {
							if _, ok := field.Type.(*ast.FuncType); ok {
								makeTag(inputText, field.Names[0], output)
							}
						}
					}
				}
			case token.VAR, token.CONST:
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
	line := tf.Line(pos)
	end := offs + len(name.Name)
	for offs > 0 && inputText[offs-1] != '\n' {
		offs--
	}
	fmt.Fprintf(output, "\x0A%s\x7F%s\x01%d,%d", inputText[offs:end], name.Name, line, offs)
}

// IdentCharSet is also used by the testing code.  The intent here is to match Go's syntax though
// without distinguishing between the initial and subsequent characters.

const identCharSet = `(?:\pL|\pN)`

// EtagsRe is not entirely etags-equivalent.  It requires the keyword to start in column 0, which is
// more limiting, but acceptable because that follows standard Go formatting for globals.  On the
// positive side it also includes var/const definitions found in column 0, won't typically include
// types defined inside functions, and it handles type parameters.
//
// Like etags, however, it won't find var/const/type definitions inside lists or subsequent
// var/const in a single definition, and it will be confused by code inside multi-line strings.

var etagsRe = regexp.MustCompile(`^(?:((?:package|func(?:\s*\([^)]+\))?|type|var|const)\s+(` + identCharSet + `+)))`)

// Note we have no file offsets.  We could fix that.

func builtinEtags(inputFn, inputText string, output io.Writer) {
	if verbose {
		log.Printf("Builtin etags: %s", inputFn)
	}
	lineno := 0
	for _, l := range strings.Split(inputText, "\n") {
		if m := etagsRe.FindStringSubmatch(l); m != nil {
			fmt.Fprintf(output, "\x0A%s\x7F%s\x01%d,", m[1], m[2], lineno+1)
		}
		lineno++
	}
}

func systemEtags(names []string, output io.Writer) {
	if verbose {
		for _, inputFn := range names {
			log.Printf("System etags: %s", inputFn)
		}
	}
	cmd := exec.Command(systemEtagsCommand, "-o", "-", "-")
	cmd.Stdin = strings.NewReader(strings.Join(names, "\n"))
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	errText := stderr.String()
	if errText != "" {
		for _, line := range strings.Split(stderr.String(), "\n") {
			log.Print(line)
		}
	}
	fmt.Fprint(output, stdout.String())
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			os.Exit(exitErr.ExitCode())
		}
	}
}
