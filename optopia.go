// Copyright 2019 Garrett D'Amore <garrett@damore.org>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Package optopia implements a fairly simple options style
// parser.  It supports long (--option) and short (-o) options.
// The reason for its existence is that we wanted something
// simple, but with support for callback functions.
package optopia

import (
	"encoding"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type err string

func (e err) Error() string {
	return string(e)
}

func (e err) Is(target error) bool {
	return strings.HasPrefix(target.Error(), string(e))
}

func mkErr(e error, opt string) err {
	return err(fmt.Sprintf("%v: %s", e, opt))
}

// These are standard error codes.
const (
	ErrNoSuchOption        = err("no such option")
	ErrOptionRequiresValue = err("option requires value")
	ErrParsingValue        = err("failure parsing option value")
	ErrDuplicateOption     = err("duplicate option")
	ErrShortAndLongEmpty   = err("long and short options both empty")
)

// Option represents a single option.  Allocate one of these and
// pass it to Options.Add() to register.
type Option struct {
	// Long is the long form of the option (without the --).
	Long string

	// Short is the short (single character) form of the option.
	Short rune

	// HasArg indicates that the option takes a value.
	// This is presumed if ArgP is not nil.
	HasArg bool

	// ArgName is the name of the associated argument.
	// Used principally in help output.
	ArgName string

	// ArgP is used to store the value.  At present
	// this can be a pointer to string, int, int64, uint64, or bool.
	// It can also be a TextUnmarshaller.
	ArgP interface{}

	// Handle is executed when this option is found, and passed the
	// raw string.  If ArgP is set, then any conversion is
	// is done first.  (If the conversion fails, then that error
	// is returned to the caller, and Handle is not called.)
	Handle func(string) error

	// Help is a short help message about the option.
	Help string

	// Seen is updated after Options.Parse.  It is true if the option
	// was seen.  This is useful for options that have no value.
	Seen bool

	// Raw contains the raw value for options that take one.
	// It is updated on Options.Parse.
	Raw string
}

// Options are the main set of Options for a program.  The zero value is
// usable immediately.
type Options struct {
	shortOpts map[rune]*Option
	longOpts map[string]*Option
	initOnce sync.Once
}

func (o *Options) init() {
	o.initOnce.Do(func() {
		o.shortOpts = make(map[rune]*Option)
		o.longOpts = make(map[string]*Option)
	})
}

// Add registers a given function.
func (o *Options) Add(opts ...*Option) error {
	o.init()
	for _, opt := range opts {
		if opt.ArgP != nil {
			opt.HasArg = true
		}
		if opt.Long == "" && opt.Short == 0 {
			return ErrShortAndLongEmpty
		}
		if opt.Long != "" {
			if o.longOpts[opt.Long] != nil {
				return mkErr(ErrDuplicateOption, string(opt.Short))
			}
			o.longOpts[opt.Long] = opt
		}
		if opt.Short != 0 {
			if o.shortOpts[opt.Short] != nil {
				return mkErr(ErrDuplicateOption, string(opt.Short))
			}
			o.shortOpts[opt.Short] = opt
		}
		opt.Seen = false
		opt.Raw = ""
	}
	return nil
}

// Reset resets the values of any Option that has been added.
// Use it to run through the option parsing multiple times.
func (o *Options) Reset() {
	o.init()
	for _, opt := range o.longOpts {
		opt.Seen = false
		opt.Raw = ""
	}
	for _, opt := range o.shortOpts {
		opt.Seen = false
		opt.Raw = ""
	}

}

// Parse parses the options. Any residual options are returned,
// and if a parse error that is returned too.
func (o *Options) Parse(args []string) ([]string, error) {
	o.init()
	for len(args) > 0 {
		arg := args[0]
		var opt *Option
		if arg == "--" {
			// End of options.
			args = args[1:]
			break
		}
		if !strings.HasPrefix(arg, "-") {
			break
		}
		if strings.HasPrefix(arg, "--") {
			// longOpts form.  First look for an exact match.
			name := strings.TrimPrefix(arg, "--")
			if opt = o.longOpts[name]; opt != nil {
				args = args[1:]
			} else {
				// Maybe its a --option=value form.  Try
				// splitting, but verify that the option
				// takes an argument.
				words := strings.SplitN(name, "=", 2)
				if len(words) == 2 {
					opt = o.longOpts[words[0]]
					if opt != nil && opt.HasArg {
						args[0] = words[1]
					} else {
						opt = nil
					}
				}
			}
		} else {
			// Starts with "-"
			name := []rune(arg[1:])
			opt = o.shortOpts[name[0]]
			if opt != nil {
				if len(name) > 1 {
					if opt.HasArg {
						// Look for -v= form. This isn't POSIX compliant.
						// If '=' is short option, then we don't do this.
						if name[1] == '=' && o.shortOpts['='] == nil {
							args[0] = string(name[2:])
						} else {
							args[0] = string(name[1:])
						}
					} else {
						// Clustered option.
						args[0] = "-" + string(name[1:])
					}
				} else {
					args = args[1:]
				}
			}
		}
		if opt == nil {
			return nil, mkErr(ErrNoSuchOption, arg)
		}

		if opt.HasArg && len(args) == 0 {
			return nil, mkErr(ErrOptionRequiresValue, arg)
		}

		val := ""
		if opt.HasArg {
			val = args[0]
			args = args[1:]
		}

		opt.Seen = true
		var e error
		if opt.HasArg {
			opt.Raw = val
		}
		if opt.HasArg && opt.ArgP != nil {
			switch v := opt.ArgP.(type) {
			case *bool:
				// we get 1, 0, true, false variants,
				// but not yes and no. We want them.
				switch val {
				case "y", "Y", "YES", "yes":
					val = "true"
				case "n", "N", "NO", "no":
					val = "false"
				}
				*v, e = strconv.ParseBool(val)
				if e != nil {
					return nil, mkErr(ErrParsingValue, arg)
				}
			case *string:
				*v = val
			case *int:
				i, e := strconv.ParseInt(val, 10, 32)
				if e != nil {
					return nil, mkErr(ErrParsingValue, arg)
				}
				*v = int(i)
			case *int64:
				*v, e = strconv.ParseInt(val, 10, 64)
				if e != nil {
					return nil, mkErr(ErrParsingValue, arg)
				}
			case *uint64:
				*v, e = strconv.ParseUint(val, 0, 64)
				if e != nil {
					return nil, mkErr(ErrParsingValue, arg)
				}
			case encoding.TextUnmarshaler:
				if e = v.UnmarshalText([]byte(val)); e != nil {
					return nil, mkErr(ErrParsingValue, arg)
				}
			}
		}

		// Handle is only run after doing any type verification.
		if opt.Handle != nil {
			if e := opt.Handle(val); e != nil {
				return nil, e
			}
			continue
		}
	}
	return args, nil
}
