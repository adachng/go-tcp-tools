// MIT License
//
// Copyright (c) 2026-present adachng (github.com/adachng)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package [cli] implements deterministic command line input parsing
// with strict input validation. This package is meant to by much more lightweight
// compared to alternatives.
//
// It is recommended to drop all references to this package's types and
// do [runtime.GC] after the [App.Parse] to reduce memory bandwidth of the program.
//
// Upon invalid usage of this package's functions, the package will panic.
//
// Upon usage of "-h" or "--help", or invalid command line input
// as configured by valid usage of this package's functions,
// [App.Parse] will return [false] as to indicate the program must not proceed and should return.
//
// The following invalid usage of [cli] functions are:
//
//   - Adding the same flag/option name (e.g. adding "-a" more than once).
//   - Adding a different flag/option name to the same value pointer. The [App] structure keeps track of a pool of pointers to ensure there is no duplicate pointer.
//   - Adding invalid flag/option name (e.g. "---address" or "--@ddress") validated by [getName].
package cli

import (
	"errors"
	"strings"
)

type parseRule int

const (
	rbool parseRule = iota

	rstring

	rint
	rint8
	rint16
	rint32
	rint64

	ruint
	ruint8
	ruint16
	ruint32
	ruint64

	rfloat32
	rfloat64
)

type validationRule int

// const (
// 	vIPv4 validationRule = iota
// 	vfileName
// )

type base struct {
	desc string
	app  *App

	val  any
	rule parseRule

	isRequired bool
	isParsed   bool
}

type Option struct {
	b base // booleans are not allowed
}

type Flag struct {
	b base // string and floats not allowed
}

type App struct {
	next *App // subcommands
	prev *App // to go back to root instance (for future use)

	// Used for parsing and then assigning the values.
	sOpts  map[string]*Option
	lOpts  map[string]*Option
	sFlags map[string]*Flag
	lFlags map[string]*Flag

	nameReg []string // strings of both short and long forms for validation against duplicate name

	ptrReg []any // registry of pointers to check against duplicate option value
}

func New() *App {
	return &App{
		next: nil,
		prev: nil,

		sOpts:  make(map[string]*Option),
		lOpts:  make(map[string]*Option),
		sFlags: make(map[string]*Flag),
		lFlags: make(map[string]*Flag),

		nameReg: []string{},
		ptrReg:  []any{},
	}
}

func isRootApp(a *App) bool {
	return a.prev == nil
}

// Used while parsing to check if random irrelevant string is inputted.
func (a *App) containsName(n string) bool {
	for i := 0; i < len(a.nameReg); i++ {
		if a.nameReg[i] == n {
			return true
		}
	}
	return false
}

// Validate and extract name string to be in the form of "-s,--string"
// with flexible spaces. Keep all validations regarding name within [getName].
//
// Its first return is the short form name (e.g. "-s") while its
// second return is the long form name (e.g. "--string").
func (a *App) getName(value any, n string) (string, string, error) {
	// Validate that value pointer is not nil.
	if value == nil {
		err := errors.New("cli: value pointer is nil")
		return "", "", err
	}

	// Validate that the value pointer is not a duplicate.
	for i := 0; i < len(a.ptrReg); i++ {
		if value == a.ptrReg[i] {
			err := errors.New("cli: value pointer already registered")
			return "", "", err
		}
	}

	n = strings.ReplaceAll(n, " ", "")

	// Empty name(s) provided.
	if len(n) <= 0 {
		return "", "", errors.New("cli: name string is empty")
	}

	strs := strings.Split(n, ",")

	// Too many names for the option/flag value.
	if len(strs) > 2 {
		return "", "", errors.New("cli: name strings consists of too many forms")
	}

	// Name too short.
	if len(strs[0]) < 2 {
		return "", "", errors.New("cli: name string too short")
	}

	// Second (if applicable) name too short.
	if len(strs) > 1 && len(strs[1]) < 2 {
		return "", "", errors.New("cli: name string too short")
	}

	var short string = ""
	var long string = ""

	// Determine which one is short form and long form if there are two forms.
	if len(strs) == 2 {
		str1 := strs[0]
		str2 := strs[1]

		if str1[0] == '-' && str1[1] == '-' {
			long = str1
			short = str2
		} else {
			short = str1
			long = str2
		}
	} else {
		// Determine if the only one is short form or long form.
		str := strs[0]
		if str[0] == '-' && str[1] == '-' {
			long = str
		} else {
			short = str
		}
	}

	// Validate short form name if short form name is present.
	if short != "" {
		if short[0] != '-' {
			return "", "", errors.New("cli: short name string does not begin with \"-\"")
		}
		for i := 1; i < len(short); i++ {
			if short[i] < 'a' && short[i] > 'z' {
				return "", "", errors.New("cli: name string contains invalid character(s)")
			}
		}
	}

	// Validate long form name if long form name is present.
	if long != "" {
		if long[0] != '-' || long[1] != '-' {
			return "", "", errors.New("cli: long name string does not begin with \"--\"")
		}
		for i := 2; i < len(long); i++ {
			if long[i] < 'a' && long[i] > 'z' {
				return "", "", errors.New("cli: name string contains invalid character(s)")
			}
		}
	}

	// Validate for duplication in registry:
	err := errors.New("cli: option/flag name already registered")
	for i := 0; i < len(a.nameReg); i++ {
		if short != "" && short == a.nameReg[i] {
			return "", "", err
		}
		if long != "" && long == a.nameReg[i] {
			return "", "", err
		}
	}

	return short, long, nil
}

// Adds pointer and short + long form names to registry (only for subsequent validation).
func (a *App) addToReg(ptr any, short string, long string) {
	a.ptrReg = append(a.ptrReg, ptr)

	// If short form name present, add to registry.
	if short != "" {
		a.nameReg = append(a.nameReg, short)
	}

	// If long form name present, add to registry.
	if long != "" {
		a.nameReg = append(a.nameReg, long)
	}
}

// Returns pointer to a new [Option] and adds the [Option] to the map for parsing.
func (a *App) newOpt(desc string, val any, r parseRule, s string, l string) *Option {
	ret := &Option{
		b: base{
			desc: desc,
			app:  a,

			val:  val,
			rule: r,

			isRequired: false,
			isParsed:   false,
		},
	}

	// If short form name is present, add it to [App.sOpts] for later parsing.
	if s != "" {
		a.sOpts[s] = ret
	}

	// If long form name is present, add it to [App.lOpts] for later parsing.
	if l != "" {
		a.lOpts[l] = ret
	}

	return ret
}

// Returns pointer to a new [Flag] and adds the [Flag] to the map for parsing.
func (a *App) newFlag(desc string, val any, r parseRule, s string, l string) *Flag {
	ret := &Flag{
		b: base{
			desc: desc,
			app:  a,

			val:  val,
			rule: r,

			isRequired: false,
			isParsed:   false,
		},
	}

	// If short form name is present, add it to [App.sFlags] for later parsing.
	if s != "" {
		a.sFlags[s] = ret
	}

	// If long form name is present, add it to [App.lFlags] for later parsing.
	if l != "" {
		a.lFlags[l] = ret
	}

	return ret
}

func (a *App) AddOptString(name string, value *string, description string) (*Option, error) {
	// Do validations and then extract the name(s).
	short, long, err := a.getName(value, name)
	if err != nil {
		// Name is invalid. Programmer misuse, not app user, so panic.
		panic(err)
		return nil, err
	}

	a.addToReg(value, short, long)

	ret := a.newOpt(description, value, rstring, short, long)
	return ret, nil
}

func (o *Option) Required() *Option {
	o.b.isRequired = true
	return o
}

// Parse the command line options. Expect passing os.Args.
//
// Ignores first string because it's the program name.
//
// Returns false if the program must return.
// Returns true if the program can proceed.
func (a *App) Parse(args []string) (bool, error) {
	if len(args) <= 1 {
		return false, errors.New("cli: not enough arguments")
	}

	// Skip the program name.
	args = args[1:]
	// TODO: handle "-h" or "--help"
	// TODO: handle "--version"
	return true, nil
}
