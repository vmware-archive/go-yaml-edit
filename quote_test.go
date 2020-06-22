// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"fmt"
	"testing"
)

func TestYamlString(t *testing.T) {
	testCases := []struct {
		src  string
		want string
	}{
		{"a", "a"},
		{"@a", "'@a'"},
		{"a#b", "a#b"},
		{"a #b", "'a #b'"},
		{"a\n", "|\n  a"},
		{"a\n\n", "|+\n  a\n"},
		{"a\nb\n", "|\n  a\n  b"},
		{"a\nb", "|-\n  a\n  b"},
		{"1", `"1"`},
		{"1.0", `"1.0"`},
		{"1.0.0", `1.0.0`},
		{"1a", `1a`},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := yamlString(tc.src, 2)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	testCases := []struct {
		src    string
		old    string
		want   string
		indent int
	}{
		{"a", "b", "a", 0},
		{"a", `"b"`, `"a"`, 0},
		{"1", "b", `"1"`, 0},
		{"1.0", "b", `"1.0"`, 0},
		{"1.0.0", "b", "1.0.0", 0},
		{"1.0.0", `"b"`, `"1.0.0"`, 0},
		{"1.0.0", `"1"`, `1.0.0`, 0},

		{"a", "'b'", "'a'", 0},
		{"a", "'#a'", "a", 0},
		{"a\nb", "'b'", "|-\n  a\n  b", 0},

		{"x: y\nbar: y\n", "|\n  x: y\nbar: x\n", "|\n  x: y\n  bar: y", 0},
		{"x: y\nbar: y\n", "|\n    x: y\n    bar: x\n", "|\n    x: y\n    bar: y", 2},
		{"bar: y\n", "|\nbar: x\n", "|\n  bar: y", 0},
		{"bar: y\n", "|\n    bar: x\n", "|\n    bar: y", 2},

		{`a`, `""`, `"a"`, 0},
		{`a`, `''`, `'a'`, 0},

		{"1", "0", `1`, 0},
		{"true", "false", `true`, 0},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := quote(tc.src, tc.old, tc.indent)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestSingleQuoted(t *testing.T) {
	testCases := []struct {
		src  string
		want string
	}{
		{"a", "'a'"},
		{`a\nb`, `'a\nb'`},
		{"a\nb", "|-\n  a\n  b"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := yamlStringTrySingleQuoted(tc.src, 2)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
