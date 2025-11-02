# gotags - reliable etags-style tag information for Go

Gotags generates an etags-like tag file for Go source, with better Go awareness
than etags.

Input file names are provided on the command line. If the only input file name
is given as "-" then the names of input files are read from standard input,
one name per line.

Input files with extension other than .go are processed by the native etags into
the specified output file.

Usage:

    gotags [flags] input-filename ...

The flags are:

    -o output-filename
        Write output to output-filename rather than to TAGS.  If output-filename
        is "-" then write to standard output.

    --etags pathname
        The name of the native etags command if not /usr/bin/etags, or specify
        the empty string to disable the use of native etags for non-Go files.

Tags are generated for all Go global names: packages, types, constants,
functions, and variables, irrespective of the declaration syntax. In contrast,
etags does not handle constants or variables, nor types defined inside type
lists, nor functions or types with type parameters, and it can mistake local
type declarations for global ones.

For full functionality, gotags requires each Go input file to be syntactically
well-formed in the sense of "go/parser". If a .go file cannot be parsed,
gotags prints a warning and falls back to its own etags-style parsing.

Input file names are emitted verbatim in the output, gotags has no resolution of
relative file names wrt the location of the output file as in etags, nor has it
support for other exotic etags functionality, such as compressed files.

Files that are passed to the native etags are processed entirely according to
etags's semantics.

To use gotags with Emacs's etags-regen-mode it is sufficient to set
etags-program-name to "gotags" in your .emacs. Note however that gotags does not
yet respect any regular expression settings in that mode for any language.
