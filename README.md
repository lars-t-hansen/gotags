# gotags - better etags-style tags for Go and Python

Gotags generates an etags-like tag file for Go and Python source, with better Go
and Python awareness than etags.

Input file names are provided on the command line. If the only input file name
is given as "-" then the names of input files are read from standard input,
one name per line.

Input files with extension other than .go are processed by the native etags into
the specified output file.

Usage:

	gotags [options] input-filename ...

Input-filename can be "-" to denote that filenames will be read from stdin.

Options:

	-h, --help
		Print usage summary
	-o filename
		`Filename` of output file, "-" for stdout, default "TAGS"
	-q, --quiet
		Suppress most warnings
	-v, --verbose
		Enable verbose output (for debugging)
	-V, --version
		Print version information
	--etags filename
		`Filename` of the native etags program, "" to disable this functionality,
		default "/usr/bin/etags"
	--no-members
		Do not tag member variables

Tags are generated for all Go global names: packages, types, constants,
functions, variables, and members of global interfaces and structs, irrespective
of the declaration syntax. In contrast, etags does not handle constants or
variables, nor types defined inside type lists, nor functions or types with
type parameters, nor interface or struct members, and it can mistake local type
declarations for global ones.

For full Go functionality, gotags requires each Go input file to be
syntactically well-formed in the sense of "go/parser". If a .go file cannot be
parsed, gotags prints a warning and falls back to its own etags-style parsing.

Tags are generated for Python function and class definitions. This uses
etags-style parsing but with better patterns than etags.

Input file names are emitted verbatim in the output, gotags has no resolution of
relative file names wrt the location of the output file as in etags, nor has it
support for other exotic etags functionality, such as compressed files.

Files that are passed to the native etags are processed entirely according to
etags's semantics.

To use gotags with Emacs's etags-regen-mode or complete-symbol it is sufficient
to set etags-program-name to "gotags" in your .emacs. Note however that gotags
does not yet respect any regular expression settings in that mode for any
language.
