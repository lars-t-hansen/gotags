# gotags - reliable etags-style tag information for Go

Gotags generates an etags-like tag file for Go source, but with better Go
awareness than etags.

Input file names are provided on the command line. If the only input file name
is given as "-" then the names of input files are read from standard input,
one name per line.

Usage:

    gotags [flags] input-filename ...

The flags are:

    -o output-filename
        Write output to output-filename rather than to TAGS.  If output-filename
        is "-" then write to standard output.

Tags are generated for all global names: packages, types, constants, functions,
and variables, irrespective of the declaration syntax. (Etags does not handle
constants or variables, nor types defined inside type lists, nor functions
or types with type parameters, and it can mistake local type declarations for
global ones.)

For full functionality, gotags requires each input file to be syntactically
well-formed in the sense of "go/parser". If a file cannot be parsed, gotags
prints a warning and falls back to etags-style parsing.

Input file names are emitted verbatim in the output, there's no resolution of
relative file names wrt the location of the output file as in etags. Nor is
there support for other exotic etags functionality, such as compressed files.
