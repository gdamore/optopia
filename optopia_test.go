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

package optopia

import (
	"net"
	"runtime"
	"strconv"
	"testing"
)

func mustAdd(t *testing.T, opts *Options, opt *Option) {
	if e := opts.Add(opt); e != nil {
		_, file, line, _ := runtime.Caller(1)

		t.Fatalf("%s:%d %v", file, line, e.Error())
	}
}
func mustFailAs(t *testing.T, e error, eIs err) {
	if e == nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d expected failure, but succeeded",
			file, line)
	}
	if !eIs.Is(e) {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d wrong error type %v != %v", file, line, e, eIs)
	}
}
func mustParse(t *testing.T, opts *Options, args []string) []string {
	res, e := opts.Parse(args)
	if e != nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d parse fail: %v", file, line, e)
	}
	return res
}
func mustNotParse(t *testing.T, opts *Options, args []string) {
	res, e := opts.Parse(args)
	if e == nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d parse passed but should have failed", file, line)
	}
	if !ErrParsingValue.Is(e) {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d parse failed wrong error %v", file, line, e)
	}
	if len(res) != 0 || res != nil {
		_, file, line, _ := runtime.Caller(1)
		t.Fatalf("%s:%d non-nil result", file, line)
	}
}

func TestErr_Is(t *testing.T) {
	e := mkErr(ErrNoSuchOption, "x")
	if !ErrNoSuchOption.Is(e) {
		t.Fail()
	}
	if ErrParsingValue.Is(e) {
		t.Fail()
	}
}

func TestOptions_Add(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	mustAdd(t, opts, o)
	if opts.longOpts["verbose"] != o {
		t.Error("Didn't register long")
	}
	if opts.shortOpts['v'] != o {
		t.Error("Didn't register short")
	}
	if len(opts.longOpts) != 1 {
		t.Error("length of long wrong")
	}
	if len(opts.shortOpts) != 1 {
		t.Error("length of short wrong")
	}
	mustAdd(t, opts, &Option{
		Short: 'x',
	})

	if len(opts.shortOpts) != 2 || len(opts.longOpts) != 1 {
		t.Error("length wrong")
	}
}

func TestOptions_Add2(t *testing.T) {
	opts := &Options{}
	mustAdd(t, opts, &Option{
		Short: 'x',
	})
	mustFailAs(t, opts.Add(&Option{Short: 'x'}), ErrDuplicateOption)
}

func TestOptions_Add3(t *testing.T) {
	opts := &Options{}
	mustFailAs(t, opts.Add(&Option{}), ErrShortAndLongEmpty)
}

func TestOptions_Add4(t *testing.T) {
	opts := &Options{} // Test non-alphas
	mustAdd(t, opts, &Option{Short: 'Г'})
	mustFailAs(t, opts.Add(&Option{Short: 'Г'}), ErrDuplicateOption)
	mustAdd(t, opts, &Option{Long: "пипец"})

}

func TestOptions_Add5(t *testing.T) {
	opts := &Options{}
	mustAdd(t, opts, &Option{
		Long: "long",
	})
	mustFailAs(t, opts.Add(&Option{Long: "long"}), ErrDuplicateOption)
}

func TestOptions_Add6(t *testing.T) {
	opts := &Options{}
	e := opts.Add(&Option{Long: "bob"}, &Option{Long: "bob"})
	mustFailAs(t, e, ErrDuplicateOption)
}

func TestOptions_Reset(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
		Seen:  true,
	}
	mustAdd(t, opts, o)
	if opts.longOpts["verbose"] != o {
		t.Error("Didn't register long")
	}
	opts.Reset()
	if o.Seen {
		t.Error("Didn't clear it")
	}
}

func TestOptions_Parse(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
		Seen:  true,
	}
	mustAdd(t, opts, o)

	args, err := opts.Parse([]string{"--bogus"})
	mustFailAs(t, err, ErrNoSuchOption)
	if args != nil {
		t.Fail()
	}
}

func TestOptions_Parse2(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	mustAdd(t, opts, o)

	args, err := opts.Parse([]string{"--verbose"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 0 || args == nil {
		t.Fail()
	}
	if !o.Seen {
		t.Fail()
	}
}

func TestOptions_Parse3(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	mustAdd(t, opts, o)

	args, err := opts.Parse([]string{"--verbose", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 1 || args[0] != "extra" {
		t.Fail()
	}
	if !o.Seen {
		t.Fail()
	}
}

func TestOptions_Parse4(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	mustAdd(t, opts, o)

	args, err := opts.Parse([]string{"-v", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 1 || args[0] != "extra" {
		t.Fail()
	}
	if !o.Seen {
		t.Fail()
	}
}

func TestOptions_Parse5(t *testing.T) {
	opts := &Options{}
	o := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	mustAdd(t, opts, o)

	args, err := opts.Parse([]string{"-v", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !o.Seen {
		t.Fail()
	}
}

func TestOptions_Parse6(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short: 'x',
		Help:  "Press the X button",
	}
	oY := &Option{
		Short: 'y',
		Help:  "Press the Y button",
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-vyx", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !oV.Seen {
		t.Fail()
	}
	if !oX.Seen {
		t.Fail()
	}
	if !oY.Seen {
		t.Fail()
	}
}

func TestOptions_Parse7(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'x',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long: "y",
		Help: "Press the Y button",
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-vxy", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !oV.Seen {
		t.Fail()
	}
	if !oX.Seen {
		t.Fail()
	}
	if oX.Raw != "y" {
		t.Fail()
	}
	if oY.Seen {
		t.Fail()
	}
}

func TestOptions_Parse8(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'x',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long: "y",
		Help: "Press the Y button",
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-vx", "--y", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !oV.Seen {
		t.Fail()
	}
	if !oX.Seen {
		t.Fail()
	}
	if oX.Raw != "--y" {
		t.Fail()
	}
	if oY.Seen {
		t.Fail()
	}
}

func TestOptions_Parse9(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'x',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long: "y",
		Help: "Press the Y button",
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-vx=--y", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !oV.Seen {
		t.Fail()
	}
	if !oX.Seen {
		t.Fail()
	}
	if oX.Raw != "--y" {
		t.Fail()
	}
	if oY.Seen {
		t.Fail()
	}
}

func TestOptions_Parse10(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-vy=--y", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fail()
	}
	if !oV.Seen {
		t.Fail()
	}
	if !oX.Seen {
		t.Fail()
	}
	if oX.Raw != "--y" {
		t.Fail()
	}
	if oY.Seen {
		t.Fail()
	}
}
func TestOptions_Parse11(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"--y=abc", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if oV.Seen {
		t.Fatal("verbose seen")
	}
	if !oY.Seen {
		t.Error("long not seen")
	}
	if oY.Raw != "abc" {
		t.Error("long value wrong")
	}
	if oX.Seen {
		t.Fail()
	}
}
func TestOptions_Parse12(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"--yx=abc", "--", "--wrong", "extra"})
	mustFailAs(t, err, ErrNoSuchOption)
	if args != nil {
		t.Fail()
	}
}

func TestOptions_Parse13(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"--y"})
	mustFailAs(t, err, ErrOptionRequiresValue)
	if args != nil {
		t.Fail()
	}
}
func TestOptions_Parse14(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"-y"})
	mustFailAs(t, err, ErrOptionRequiresValue)
	if args != nil {
		t.Fail()
	}
}

func TestOptions_Parse15(t *testing.T) {
	opts := &Options{}
	oV := &Option{
		Short: 'v',
		Long:  "verbose",
		Help:  "Enable verbose output",
	}
	oX := &Option{
		Short:  'y',
		Help:   "Press the X button",
		HasArg: true,
	}
	oY := &Option{
		Long:   "y",
		Help:   "Press the Y button",
		HasArg: true,
	}
	mustAdd(t, opts, oV)
	mustAdd(t, opts, oX)
	mustAdd(t, opts, oY)

	args, err := opts.Parse([]string{"--y", "--", "-v", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oV.Seen {
		t.Fatal("verbose seen")
	}
	if !oY.Seen {
		t.Error("long not seen")
	}
	if oY.Raw != "--" {
		t.Error("long value wrong")
	}
	if oX.Seen {
		t.Fail()
	}
}

func TestOptions_Parse16(t *testing.T) {
	opts := &Options{}
	var val string
	oX := &Option{
		Long:   "y",
		Help:   "Press the X button",
		HasArg: true,
		ArgP:   &val,
	}
	mustAdd(t, opts, oX)

	args, err := opts.Parse([]string{"--y", "--", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "--" {
		t.Error("long value wrong")
	}
	if val != "--" {
		t.Error("did not store in receiver")
	}
}
func TestOptions_Parse17(t *testing.T) {
	opts := &Options{}
	var val bool
	oX := &Option{
		Long: "y",
		Help: "Press the X button",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args, err := opts.Parse([]string{"--y", "true", "--", "--wrong", "extra"})
	if err != nil {
		t.Fail()
	}
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "true" {
		t.Error("long value wrong")
	}
	if !val {
		t.Error("did not store in receiver")
	}
}
func TestOptions_Parse18(t *testing.T) {
	opts := &Options{}
	var val bool
	oX := &Option{
		Long: "y",
		Help: "Press the X button",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--y", "yes", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "yes" {
		t.Error("long value wrong")
	}
	if !val {
		t.Error("did not store in receiver")
	}

	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "YES"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "Y"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "y"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "NO"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "no"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "N"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "n"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "1"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "t"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "0"})
	opts.Reset()
	_ = mustParse(t, opts, []string{"--y", "f"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--y", "2"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--y", "bogus"})

}

func TestOptions_Parse19(t *testing.T) {
	opts := &Options{}
	var val int
	oX := &Option{
		Long: "x",
		Help: "signed X coordinate",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--x", "32", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "32" {
		t.Error("long value wrong")
	}
	if val != 32 {
		t.Error("did not store in receiver")
	}

	opts.Reset()
	_ = mustParse(t, opts, []string{"--x", "-1023"})
	if val != -1023 {
		t.FailNow()
	}
	opts.Reset()

	mustNotParse(t, opts, []string{"--x", "2.5"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", ""})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "0x100"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "1000000000000"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "Junk"})

}

func TestOptions_Parse20(t *testing.T) {
	opts := &Options{}
	var val int64
	oX := &Option{
		Long: "x",
		Help: "signed X coordinate",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--x", "32", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "32" {
		t.Error("long value wrong")
	}
	if val != 32 {
		t.Error("did not store in receiver")
	}

	opts.Reset()
	_ = mustParse(t, opts, []string{"--x", "-1023"})
	if val != -1023 {
		t.FailNow()
	}
	opts.Reset()
	_ = mustParse(t, opts, []string{"--x", "1000000000000"})
	if val != 1000000000000 {
		t.FailNow()
	}
	opts.Reset()

	mustNotParse(t, opts, []string{"--x", "2.5"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", ""})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "0x100"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "JUNK"})

}

func TestOptions_Parse21(t *testing.T) {
	opts := &Options{}
	var val uint64
	oX := &Option{
		Long: "x",
		Help: "unsigned X coordinate",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--x", "32", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "32" {
		t.Error("long value wrong")
	}
	if val != 32 {
		t.Error("did not store in receiver")
	}

	opts.Reset()
	_ = mustParse(t, opts, []string{"--x", "0x100"})
	if val != 256 {
		t.FailNow()
	}
	opts.Reset()
	_ = mustParse(t, opts, []string{"--x", "1000000000000"})
	if val != 1000000000000 {
		t.FailNow()
	}
	opts.Reset()

	mustNotParse(t, opts, []string{"--x", "2.5"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", ""})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "-123"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--x", "JUNK"})
}

func TestOptions_Parse22(t *testing.T) {
	opts := &Options{}
	var val net.IP
	oX := &Option{
		Long: "ip",
		Help: "ip address",
		ArgP: &val,
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--ip", "8.8.8.8", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("long not seen")
	}
	if oX.Raw != "8.8.8.8" {
		t.Error("long value wrong")
	}
	if val.String() != "8.8.8.8" {
		t.Error("did not store in receiver")
	}

	opts.Reset()
	_ = mustParse(t, opts, []string{"--ip", "001.002.3.04"})
	if val.String() != "1.2.3.4" {
		t.FailNow()
	}
	opts.Reset()
	_ = mustParse(t, opts, []string{"--ip", "127.0.0.1"})
	if val.IsMulticast() || !val.IsLoopback() {
		t.FailNow()
	}
	opts.Reset()
	_ = mustParse(t, opts, []string{"--ip", "::1"})
	opts.Reset()
	if val.IsMulticast() || !val.IsLoopback() {
		t.FailNow()
	}

	mustNotParse(t, opts, []string{"--ip", "2.5"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--ip", "localhost"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--ip", "-123"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--ip", "JUNK"})
}

func TestOptions_Parse23(t *testing.T) {
	opts := &Options{}
	var val int
	var val2 int
	oX := &Option{
		Long: "i",
		Help: "i",
		ArgP: &val,
		Handle: func(s string) error {
			i, e := strconv.Atoi(s)
			if e != nil {
				return e
			}
			if i % 2 != 0 {
				return err("even numbers only")
			}
			val2 = i
			return nil
		},
	}
	mustAdd(t, opts, oX)

	args := mustParse(t, opts, []string{"--i", "32", "--", "--wrong", "extra"})
	if len(args) != 2 || args[0] != "--wrong" || args[1] != "extra" {
		t.Fatal("oops")
	}
	if !oX.Seen {
		t.Error("not seen")
	}
	if oX.Raw != "32" {
		t.Error("long value wrong")
	}
	if val2 != 32 {
		t.Error("didn't get our value")
	}
	if val != 32 {
		t.Error("neither did common code")
	}
	opts.Reset()

	mustNotParse(t, opts, []string{"--i", "2.5"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--i", "localhost"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--i", "0x123"})
	opts.Reset()
	mustNotParse(t, opts, []string{"--i", "JUNK"})

	_, e := opts.Parse([]string{"--i=3"})
	if e == nil || e.Error() != "even numbers only" {
		t.Errorf("handler didn't fail")
	}
}
