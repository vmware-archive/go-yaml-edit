// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package splice_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mkmik/filetransformer"
	"github.com/vmware-labs/go-yaml-edit/splice"
	"golang.org/x/text/transform"
)

func ExampleOp() {
	fmt.Printf("%T", splice.Span(3, 4).With("foo"))
	// Output:
	// splice.Op
}

// In order to splice a string you must provide one or more spans (character ranges)
// and the replacement string for each span.
// The resulting Transformer instance can then be passed to a collection of functions
// that can apply transformers to their inputs.
// See the documentation for the golang.org/x/text/transform for a full list.
//
// This example shows the simplest case where the input and output are plain Go strings.
func Example() {
	src := "abcd"

	res, _, err := transform.String(splice.T(splice.Span(1, 2).With("XYZ")), src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// aXYZcd
}

// All positions are character (rune) position, not bytes. This is an important
// distinction when the input contains non-ASCII characters.
func Example_unicode() {
	src := "ábcd"

	res, _, err := transform.String(splice.T(splice.Span(1, 2).With("B")), src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// áBcd
}

// Multiple substitutions are possible. All positions are interpreted as positions in the source stream.
// The library deals with the fact that replacement strings can have different lengths than the strings
// they replace and will behave accordingly.
func Example_multiple() {
	src := "abcd"

	t := splice.T(
		splice.Span(1, 2).With("B"),
		splice.Span(3, 4).With("D"),
	)
	aBcD, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBcD)

	t = splice.T(
		splice.Span(1, 2).With("Ba"),
		splice.Span(2, 3).With(""),
		splice.Span(3, 4).With("Da"),
	)
	aBaDa, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBaDa)

	// Output:
	// aBcD
	// aBaDa
}

// Inserting text is achieved by selecting a zero-width span, effectively acting
// as a cursor inside the input stream.
func Example_insert() {
	src := "abcd"

	t := splice.T(splice.Span(2, 2).With("X"))
	res, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abXcd
}

// Deletion is modeled by using an empty replacement string.
func Example_delete() {
	src := "abcd"

	t := splice.T(splice.Span(2, 3).With(""))
	res, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abd
}

// Applying a YAML edit on a byte slice can be achieved by applying a transformer
// on an input byte slice using the transform.Bytes function of the
// golang.org/x/text/transform package.
func ExampleBytes() {
	buf := []byte("abcd")

	t := splice.T(splice.Span(1, 2).With("B"))
	aBcd, _, err := transform.Bytes(t, buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(aBcd))

	// Output:
	// aBcd
}

// If you want to edit a file in-place, just use any library that can apply a Transformer
// to a file, like for example the github.com/mkmik/filetransformer package.
func Example_file() {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "abcd")
	tmp.Close()

	t := splice.T(splice.Span(1, 3).With("X"))
	if err := filetransformer.Transform(t, tmp.Name()); err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// aXd
}

func TestOps(t *testing.T) {
	rep := func(ops ...splice.Op) []splice.Op { return ops }
	testCases := []struct {
		in   string
		want string
		ops  []splice.Op
	}{
		{"abcd", "abXcd", rep(splice.Span(2, 2).With("X"))},
		{"abcd", "abd", rep(splice.Span(2, 3).With(""))},
		{"abcd", "abYd", rep(splice.Span(2, 3).With("Y"))},
		{"abcd", "ab x d", rep(splice.Span(2, 3).With(" x "))},
		{"ab x d", "abcd", rep(splice.Span(2, 5).With("c"))},
		{"abcd", "abcd$", rep(splice.Span(4, 4).With("$"))},
		{"abcd", "^abcd", rep(splice.Span(0, 0).With("^"))},
		{"abcd", "", rep(splice.Span(0, 4).With(""))},
		{"", "abcd", rep(splice.Span(0, 0).With("abcd"))},
		{
			"abcde xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx from xyz",
			"aBCdE xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx from xyz",
			rep(
				splice.Span(1, 2).With("B"),
				splice.Span(2, 3).With("C"),
				splice.Span(4, 5).With("E"),
			),
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			tr := splice.T(tc.ops...)
			got, _, err := transform.String(tr, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestPeek(t *testing.T) {
	s := func(s ...splice.Selection) []splice.Selection { return s }
	testCases := []struct {
		in   string
		sel  []splice.Selection
		want []string
	}{
		{"abcd", s(splice.Span(1, 2)), []string{"b"}},
		{"abcd", s(splice.Span(1, 2), splice.Span(2, 3)), []string{"b", "c"}},
		{"abcd", s(splice.Span(1, 3)), []string{"bc"}},
		{"abcd", s(splice.Span(0, 4)), []string{"abcd"}},
		{"abcd", s(splice.Span(3, 4)), []string{"d"}},
		{"abcd", s(splice.Span(4, 4)), []string{""}},
		{"abcd", s(splice.Span(1, 3), splice.Span(3, 4)), []string{"bc", "d"}},
		{"abcd", s(splice.Span(3, 4), splice.Span(1, 3)), []string{"d", "bc"}},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := splice.Peek(strings.NewReader(tc.in), tc.sel...)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
