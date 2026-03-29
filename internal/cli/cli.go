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

// This package does not support concurrency.
package cli

import (
	"errors"
	"fmt"
	"strings"
)

type parseRule int

const (
	Rbool parseRule = iota

	Rstring

	Rint
	Rint8
	Rint16
	Rint32
	Rint64

	Ruint
	Ruint8
	Ruint16
	Ruint32
	Ruint64

	Rfloat32
	Rfloat64
)

type name struct {
	shortForm string
	longForm  string
}

type base struct {
	n    name
	desc string

	val  any
	rule parseRule

	isRequired bool
	isParsed   bool
}

type Option struct {
	common base
}

type Flag struct {
	common base // string and floats not allowed
}

type App struct {
	opts  map[name]Option
	flags map[name]Flag
}

func New() App {
	return App{
		opts:  make(map[name]Option),
		flags: make(map[name]Flag),
	}
}

func (a *App) AddOptString(name string,
	value *string,
	description string) (*Option, error) {

	if value == nil {
		return nil, errors.New("cli: nil pointer passed in as value")
	}

	if len(name) == 0 {
		return nil, errors.New("cli: option name empty")
	}

	name = strings.ReplaceAll(name, " ", "")
	strs := strings.Split(name, ",")

	fmt.Print(strs)

	// TODO: separate short form and long form
	// put into name

	// if name wrong, return nil, errors.New("cli: invalid option name")

	// TODO: validate name, must be in form of something like "-a, --address"
	//

	ret := &Option{
		common: base{
			// n : name {},
			desc: description,
			val:  value,

			rule: Rstring,

			isRequired: false,
			isParsed:   false,
		},
	}
	return ret, nil
}

func (o *Option) Required() *Option {
	o.common.isRequired = true
	return o
}

// Parse the command line options. Expect passing os.Args.
//
// Ignores first string because it's the program name.
func (a *App) Parse(args []string) error {
	return errors.New("AAA")
}
