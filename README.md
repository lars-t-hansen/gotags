Generate Emacs etags-style tag information for Go.  Information is generated for packages,
functions, global types, global variables, and global constants. If a file is not syntactically
well-formed, a warning is emitted.

Unlike etags, gotags distinguishes between local and global types, can handle declaration lists and
type parameters, and handles variables and constants.

Normally you would run this on a list of input files and the output file would default to TAGS; use
-o to override the output name.
