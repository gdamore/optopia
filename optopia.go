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
	if strings.HasPrefix(target.Error(), string(e)) {
		return true
	}
	return false
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
	ErrShortOptionTooLong  = err("short option too long")
	ErrShortAndLongEmpty   = err("long and short options both empty")
)

// Option represents a single option.  Allocate one of these and
// pass it to Options.Add() to register.
type Option struct {
	// Long is the long form of the option (without the --).
	Long string

	// Short is the short (single character) form of the option.
	Short string

	// HasValue indicates that the option takes a value.
	// This setting only matters if Value is nil and Handle
	// is not.  (Note that either Value should be non-nil, or
	// Handle should.)
	HasValue bool

	// Value is used to store the value.  At present
	// this can be an integer or a string.
	ValueReceiver interface{}

	// ValueName is the name of the associated argument.
	// Used principally in help output.
	ValueName string

	// Handle is executed when this option is found, and passed the
	// raw string.  If ValueReceiver is set, then any conversion is
	// is done first.  (If the conversion fails, then that error
	// is returned to the caller, and Handle is not called.)
	Handle func(string) error

	// Description is a short help message about the option.
	Description string

	// Seen is updated after Options.Parse.  It is true if the option
	// was seen.  This is useful for options that have no value.
	Seen bool

	// RawValue contains the raw value for options that take one.
	// It is updated on Options.Parse.
	RawValue string
}

// Options are the main set of Options for a program.  The zero value is
// usable immediately.
type Options struct {
	// Short is the map of short (-o) options
	Short map[string]*Option

	// Long is the map of long (--option) options.
	Long map[string]*Option

	initOnce sync.Once
}

func (o *Options) init() {
	o.initOnce.Do(func() {
		o.Short = make(map[string]*Option)
		o.Long = make(map[string]*Option)
	})
}

// Add registers a given function.
func (o *Options) Add(opts ...*Option) error {
	o.init()
	for _, opt := range opts {
		if opt.ValueReceiver != nil {
			opt.HasValue = true
		}
		if opt.Long == "" && opt.Short == "" {
			return ErrShortAndLongEmpty
		}
		if opt.Long != "" {
			if o.Long[opt.Long] != nil {
				return mkErr(ErrDuplicateOption, opt.Short)
			}
			o.Long[opt.Long] = opt
		}
		if opt.Short != "" {
			if len(opt.Short) > 1 {
				return mkErr(ErrShortOptionTooLong, opt.Short)
			}
			if o.Short[opt.Short] != nil {
				return mkErr(ErrDuplicateOption, opt.Short)
			}
			o.Short[opt.Short] = opt
		}
		opt.Seen = false
		opt.RawValue = ""
	}
	return nil
}

// Reset resets the values of any Option that has been added.
// Use it to run through the option parsing multiple times.
func (o *Options) Reset() {
	o.init()
	for _, opt := range o.Long {
		opt.Seen = false
		opt.RawValue = ""
	}
	for _, opt := range o.Short {
		opt.Seen = false
		opt.RawValue = ""
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
			// Long form.  First look for an exact match.
			name := strings.TrimPrefix(arg, "--")
			if opt = o.Long[name]; opt != nil {
				args = args[1:]
			} else {
				// Maybe its a --option=value form.  Try
				// splitting, but verify that the option
				// takes an argument.
				words := strings.SplitN(name, "=", 2)
				if len(words) == 2 {
					opt = o.Long[words[0]]
					if opt != nil && opt.HasValue {
						args[0] = words[1]
					} else {
						opt = nil
					}
				}
			}
		} else {
			name := strings.TrimPrefix(arg, "-")
			opt = o.Short[name[:1]]
			if opt != nil {
				if len(name) > 1 {
					if opt.HasValue {
						// Look for -v= form. This isn't POSIX compliant.
						// If '=' is short option, then we don't do this.
						if name[1] == '=' && o.Short["="] == nil {
							args[0] = name[2:]
						} else {
							args[0] = name[1:]
						}
					} else {
						// Clustered option.
						args[0] = "-" + name[1:]
					}
				} else {
					args = args[1:]
				}
			}
		}
		if opt == nil {
			return nil, mkErr(ErrNoSuchOption, arg)
		}

		if opt.HasValue && len(args) == 0 {
			return nil, mkErr(ErrOptionRequiresValue, arg)
		}

		val := ""
		if opt.HasValue {
			val = args[0]
			args = args[1:]
		}

		opt.Seen = true
		var e error
		if opt.HasValue {
			opt.RawValue = val
		}
		if opt.HasValue && opt.ValueReceiver != nil {
			switch v := opt.ValueReceiver.(type) {
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
