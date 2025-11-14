// SPDX-License-Identifier: MIT
// From https://github.com/lars-t-hansen/util/go-utils

// GetOpts is a simple table-driven argument parser that is more or less compatible with GNU
// getopt-style parsing.  The parser takes a table of options and handlers, and a list of
// arguments, and parses the arguments, invoking handlers for the options encountered.
//
// Argument parsing stops at a lone '--', any remaining arguments are returned as a slice from the
// call.
//
// An argument that has no handler results in an error being returned.
//
// An option can have a long name (starting with '--') and/or a short name (starting with '-').  If
// it has neither it is the default handler, which is invoked for non-option arguments that appear
// before the lone '--'.
//
// An option can have the short name '-' (and no long name and take no value).  This is recognized
// as a special case.
//
// Long names with values can use the syntax '--opt=value' or '--opt value'.
//
// Short names with values can use the syntax '-n value' or '-nvalue', in the latter case provided
// only that the first letter of the value is not a valid short option letter.  Thus '-k 2' can be
// written '-k2'.
//
// Short names can be run together, eg '-nr' is equivalent to '-n -r'.
//
// Short names be run together with a value for at most one of the options, as in '-nkr2' for
// 'sort(1)', equivalent to -n -k 2 -r; also '-nkr 2' is accepted.  Again the boundary between
// option letters and value in the former case is where a letter is not a valid option letter, so
// '-nk2r' won't work as this is interpreted as '-n -k 2r'.
//
// Failure to validate the option table results in a panic.
//
// Any error returned from GetOpts is an error returned from the parser due to an unhandled option
// in the arguments or from one of the handlers as a validation error, the latter usually wrapped in
// a parser error explaining the context.
//
// An option is not repeatable unless its Repeatable attribute is set (including the default
// option).
//
// (An argument could be made that GetOpts should be parameterized over a context type and should
// take a context value that it passes to the handlers.  I'm not doing this because it's not
// normally needed and because using closures for handlers can accomplish the same when it is
// needed.)
//
// TODO: Looking at tail(1), option values are sometimes optional (for --follow for example).  That
// can't be expressed here and I'm not sure what the syntax would be.  For "--follow=" it's obvious,
// but for "-f" and "--follow" it is not.  Need to look at the source.

package utils

import (
	"fmt"
	"io"
	"strings"
)

type Option struct {
	Short      rune
	Long       string
	Help       string
	Value      bool
	Repeatable bool
	Handler    func(value string) error
}

// Parse args, passing argument values to the handlers of options along with the cx, and return any
// left-over arguments.
func GetOpts(options []Option, args []string) ([]string, error) {
	short, long, defaultOption := parseOptionTable(options)

	handled := make(map[*Option]bool)
	argIx := 0
	for argIx < len(args) {
		arg := args[argIx]
		argIx++
		if len(arg) > 0 && arg[0] == '-' {
			if len(arg) > 1 && arg[1] == '-' {
				if len(arg) == 2 {
					return args[argIx:], nil
				}
				arg = arg[2:]
				optname, value, matched := strings.Cut(arg, "=")
				opt := long[optname]
				if opt == nil {
					return nil, fmt.Errorf("Unknown option \"--%s\"", optname)
				}
				if !opt.Repeatable && handled[opt] {
					return nil, fmt.Errorf("Repeated but unrepeatable option \"--%s\"", optname)
				}
				handled[opt] = true
				if !opt.Value && matched {
					return nil, fmt.Errorf("Option \"--%s\" does not take a value", optname)
				}
				if opt.Value && !matched {
					if argIx == len(args) {
						return nil, fmt.Errorf("Missing value for option \"--%s\"", optname)
					}
					value = args[argIx]
					argIx++
				}
				err := opt.Handler(value)
				if err != nil {
					return nil, fmt.Errorf("Rejected option \"--%s\": %w", optname, err)
				}
			} else {
				i := 0
				if len(arg) > 1 {
					i++
				}
				var needValue *Option
				for {
					if i == len(arg) {
						break
					}
					// TODO: this is going by byte value but should we not go by rune in the option
					// string too?  Then we must also handle invalid UTF8 probably.
					optname := rune(arg[i])
					opt := short[optname]
					if opt == nil {
						if i < 2 {
							return nil, fmt.Errorf("Illegal option")
						}
						break
					}
					i++
					if !opt.Repeatable && handled[opt] {
						return nil, fmt.Errorf("Repeated but unrepeatable option \"-%c\"", optname)
					}
					handled[opt] = true
					if opt.Value {
						if needValue != nil {
							return nil, fmt.Errorf("Multiple short options compete for a value")
						}
						needValue = opt
						continue
					}
					err := opt.Handler("")
					if err != nil {
						return nil, err
					}
				}
				if needValue != nil {
					var value string
					if i < len(arg) {
						value = arg[i:]
					} else {
						if argIx == len(args) {
							return nil, fmt.Errorf("Missing value for option \"-%c\"", needValue.Short)
						}
						value = args[argIx]
						argIx++
					}
					err := needValue.Handler(value)
					if err != nil {
						return nil, err
					}
				}
			}
		} else {
			if !defaultOption.Repeatable && handled[defaultOption] {
				return nil, fmt.Errorf("Repeated but unrepeatable default option")
			}
			handled[defaultOption] = true
			err := defaultOption.Handler(arg)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func parseOptionTable(
	options []Option,
) (short map[rune]*Option, long map[string]*Option, defaultOption *Option) {
	short = make(map[rune]*Option)
	long = make(map[string]*Option)

	for optIx := range options {
		opt := &options[optIx]
		if opt.Handler == nil {
			panic("Option without handler")
		}
		if opt.Short == 0 && opt.Long == "" {
			if defaultOption != nil {
				panic("Multiple default handlers")
			}
			defaultOption = opt
		}
		if opt.Short != 0 {
			if short[opt.Short] != nil {
				panic("Multiple definitions for short option")
			}
			if opt.Short == '-' {
				if opt.Long != "" {
					panic("Long name defined for lone dash")
				}
				if opt.Value {
					panic("Lone dash cannot take a value")
				}
			}
			short[opt.Short] = opt
		}
		if opt.Long != "" {
			if long[opt.Long] != nil {
				panic("Multiple definitions for long")
			}
			long[opt.Long] = opt
		}
	}
	if defaultOption == nil {
		defaultOption = &Option{
			Handler: func(_ string) error {
				return fmt.Errorf("Additional arguments not allowed")
			},
		}
	}
	return
}

// PrintOpts prints the option names and help text of the options table in a sensible way on output,
// as for a usage message.
//
// It is printed as
//
//   -short-option, --long-option value
//     Explanation
//
// where the short and long options are omitted as expected and "value" is either that word or the
// first string in the help text that is enclosed in backticks (as for the Go flags system).  The
// options are indented two spaces, the explanation four spaces, there is no automatic wrapping of
// the text.
func PrintOpts(output io.Writer, options []Option) {
	for _, o := range options {
		if o.Help == "" || (o.Short == 0 && o.Long == "") {
			continue
		}
		otext := "  "
		if o.Short != 0 {
			otext += fmt.Sprintf("-%c", o.Short)
		}
		if o.Long != "" {
			if o.Short != 0 {
				otext += ", "
			}
			otext += fmt.Sprintf("--%s", o.Long)
		}
		if o.Value {
			vtext := "value"
			if ix := strings.Index(o.Help, "`"); ix != -1 {
				if iy := strings.Index(o.Help[ix+1:], "`"); iy != -1 {
					iy += ix+1
					if iy - ix > 1 {
						vtext = strings.ToLower(o.Help[ix+1:iy])
					}
				}
			}
			otext += " " + vtext
		}
		fmt.Fprintln(output, otext)
		fmt.Fprint(output, "    ")
		fmt.Fprintln(output, o.Help)
	}
}

// We can have more of these...

// A simple handler that will set a flag to true and always succeed
func SetFlag(flagp *bool) func(string) error {
	return func(_ string) error {
		*flagp = true
		return nil
	}
}

// A simple handler that will set a string to a given value and always succeed
func SetString(stringp *string) func(string) error {
	return func(s string) error {
		*stringp = s
		return nil
	}
}
