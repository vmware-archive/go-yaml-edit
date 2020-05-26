// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/vmware-labs/go-yaml-edit"
	yptr "github.com/vmware-labs/yaml-jsonpointer"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

// ExampleT shows how to use the transformer to edit a YAML source in place.
// It also shows how the quoting style is preserved
func ExampleT() {
	src := `apiVersion: v1
kind: Service
metadata:
  name: "foo"
  namespace: myns
`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		log.Fatal(err)
	}
	nameNode, err := yptr.Find(&root, "/metadata/name")
	if err != nil {
		log.Fatal(err)
	}
	nsNode, err := yptr.Find(&root, "/metadata/namespace")
	if err != nil {
		log.Fatal(err)
	}

	out, _, err := transform.String(yamled.T(
		yamled.Node(nameNode).With("bar"),
		yamled.Node(nsNode).With("otherns"),
	), src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)

	// Output:
	// apiVersion: v1
	// kind: Service
	// metadata:
	//   name: "bar"
	//   namespace: otherns
}

func TestEdit(t *testing.T) {
	srcs := []struct {
		src string
		foo string
		bar string
	}{
		{
			src: `foo: abc
bar: xy
baz: end
`,
			foo: "/foo",
			bar: "/bar",
		},
		{
			src: `foo: abc
data:
  bar: xy
baz: end
`,
			foo: "/foo",
			bar: "/data/bar",
		},
		{
			src: `bar: xy
data:
  foo: abc
baz: end
`,
			foo: "/data/foo",
			bar: "/bar",
		},
		{
			src: `bar: xy
data:
  deeper:
    foo: abc
baz: end
`,
			foo: "/data/deeper/foo",
			bar: "/bar",
		},
	}

	testCases := []struct {
		foo string
		bar string
	}{
		{
			foo: "AB",
			bar: "xyz",
		},
		{
			foo: "ABCD",
			bar: "x",
		},
		{
			foo: "ABCD",
			bar: "",
		},
		{
			foo: "",
			bar: "x",
		},
		{
			foo: "",
			bar: "a#b",
		},
		{
			foo: "",
			bar: "a #b",
		},
		{
			foo: "",
			bar: " ",
		},
		{
			foo: "a",
			bar: "2",
		},
		{
			foo: "a\nb\n",
			bar: "ab",
		},
		{
			foo: "\na\nb\n",
			bar: "ab",
		},
		{
			foo: "\na\nb\n\n\n",
			bar: "ab",
		},
		{
			foo: "a",
			bar: "\n",
		},
	}

	for i, tc := range testCases {
		for j, cfg := range srcs {
			t.Run(fmt.Sprintf("%d_%d", i, j), func(t *testing.T) {
				var n yaml.Node
				buf := []byte(cfg.src)
				if err := yaml.Unmarshal(buf, &n); err != nil {
					t.Fatal(err)
				}

				foo, err := yptr.Find(&n, cfg.foo)
				if err != nil {
					t.Fatal(err)
				}

				bar, err := yptr.Find(&n, cfg.bar)
				if err != nil {
					t.Fatal(err)
				}

				buf, _, err = transform.Bytes(yamled.T(
					yamled.Node(foo).With(tc.foo),
					yamled.Node(bar).With(tc.bar),
				), []byte(cfg.src))
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("after:\n%s", string(buf))

				var ne yaml.Node
				if err := yaml.Unmarshal(buf, &ne); err != nil {
					t.Fatal(err)
				}

				check := func(path, want string) {
					f, err := yptr.Find(&ne, path)
					if err != nil {
						t.Fatal(err)
					}
					if got := f.Value; got != want {
						t.Errorf("got: %q, want: %q", got, want)
					}

					if tag := f.Tag; tag != "!!str" && tag != "!!null" {
						t.Errorf("tag for %q must be either string or null, got %q", path, tag)
					}
				}
				check(cfg.foo, tc.foo)
				check(cfg.bar, tc.bar)
			})
		}
	}
}

func TestIndent(t *testing.T) {
	// the long strings here exercise ErrShortSrc and ErrShortDst codepaths in the transformer.
	src := `out:
  foo: abc
  other:
    bar: xy
baz: end
xxx: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
yyy: yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
www:
  y: zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz
`

	testCases := []struct {
		foo string
		bar string
	}{
		{
			foo: "edit test",
			bar: "long\nzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var n yaml.Node
			buf := []byte(src)
			if err := yaml.Unmarshal(buf, &n); err != nil {
				t.Fatal(err)
			}

			foo, err := yptr.Find(&n, "/out/foo")
			if err != nil {
				t.Fatal(err)
			}

			bar, err := yptr.Find(&n, "/out/other/bar")
			if err != nil {
				t.Fatal(err)
			}

			buf, _, err = transform.Bytes(yamled.T(
				yamled.Node(foo).With(tc.foo),
				yamled.Node(bar).With(tc.bar),
			), []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("after:\n%s", string(buf))

			var ne yaml.Node
			if err := yaml.Unmarshal(buf, &ne); err != nil {
				t.Fatal(err)
			}

			check := func(path, want string) {
				f, err := yptr.Find(&ne, path)
				if err != nil {
					t.Fatal(err)
				}
				if got := f.Value; got != want {
					t.Errorf("got: %q, want: %q", got, want)
				}

				if tag := f.Tag; tag != "!!str" && tag != "!!null" {
					t.Errorf("tag for %q must be either string or null, got %q", path, tag)
				}
			}
			check("/out/foo", tc.foo)
			check("/out/other/bar", tc.bar)
		})
	}
}

func TestBug1(t *testing.T) {
	src := `data:
  foo: |
    bar: x
`

	rep := `x: y
bar: y
`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatal(err)
	}
	foo, err := yptr.Find(&root, "/data/foo")
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := transform.Bytes(yamled.T(
		yamled.Node(foo).With(rep),
	), []byte(src))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("got:\n%s", out)

	want := `data:
  foo: |
    x: y
    bar: y
`
	if got := string(out); got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}
