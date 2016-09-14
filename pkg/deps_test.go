/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

package pkg

import (
	"go/build"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	// Expected import layout. All names (except stdDeps) are relative to
	// "github.com/google/godepq/testing"
	expectations = map[string][]string{
		"":         []string{"a", "b"},
		"a":        []string{"a/aa", "a/ab"},
		"a/aa":     []string{"a/aa/aaa"},
		"a/aa/aaa": nil,
		"a/ab":     nil,
		"b":        []string{"b/ba"},
		"b/ba":     nil,
	}
)

const basePkg = "github.com/google/godepq/testing"

func TestBuildBare(t *testing.T) {
	deps := testBuildBasic(t, false, false)
	assertSetsEqual(t, deps.Ignored, NewSet(Package("errors")), "Ignored")
}

func TestBuildWithStdlib(t *testing.T) {
	deps := testBuildBasic(t, true, false)
	assert.Len(t, deps.Ignored, 0)
}

func TestBuildWithTests(t *testing.T) {
	deps := testBuildBasic(t, false, true)
	assertSetsEqual(t, deps.Ignored, NewSet(Package("errors")), "Ignored")
}

func testBuildBasic(t *testing.T, includeStdlib, includeTests bool) Dependencies {
	deps, err := (&Builder{
		Roots:         []Package{Package(basePkg)},
		BuildContext:  build.Default,
		IncludeStdlib: includeStdlib,
		IncludeTests:  includeTests,
	}).Build()
	assert.NoError(t, err)
	assertGraphsEqual(t, deps.Forward, expectedGraph(includeStdlib, includeTests))
	return deps
}

func TestIgnoreBasic(t *testing.T) {
	deps, err := (&Builder{
		Roots:        []Package{Package(basePkg)},
		BuildContext: build.Default,
		Ignored: []*regexp.Regexp{
			regexp.MustCompile(basePkg + "/b.*"),
		},
	}).Build()
	assert.NoError(t, err)

	// Build expected graph without "b" packages.
	expected := expectedGraph(false, false)
	delete(expected, mkpkg("b"))
	delete(expected, mkpkg("b/ba"))
	delete(expected.Pkg(mkpkg("")), mkpkg("b"))

	assertGraphsEqual(t, deps.Forward, expected)
	expectedIgnores := NewSet(mkpkg("b"), Package("errors")) // Never reach "b/ba".
	assertSetsEqual(t, deps.Ignored, expectedIgnores, "Ignored")
}

func TestIgnoreWithStdlibAndTests(t *testing.T) {
	deps, err := (&Builder{
		Roots:        []Package{Package(basePkg)},
		BuildContext: build.Default,
		Ignored: []*regexp.Regexp{
			regexp.MustCompile("err?ors"),         // Ignore stdlib
			regexp.MustCompile("github.com/.*/c"), // Ignore tests
		},
		IncludeStdlib: true,
		IncludeTests:  true,
	}).Build()
	assert.NoError(t, err)
	assertGraphsEqual(t, deps.Forward, expectedGraph(false, false))
	expectedIgnores := NewSet(mkpkg("c"), Package("errors"))
	assertSetsEqual(t, deps.Ignored, expectedIgnores, "Ignored")
}

func TestInclude(t *testing.T) {
	deps, err := (&Builder{
		Roots:        []Package{Package(basePkg)},
		BuildContext: build.Default,
		Included: []*regexp.Regexp{
			regexp.MustCompile("^" + basePkg + "$"),
			regexp.MustCompile("github.com/.*/[ab].*"), // Ignore tests
		},
		IncludeStdlib: true,
		IncludeTests:  true,
	}).Build()
	assert.NoError(t, err)
	assertGraphsEqual(t, deps.Forward, expectedGraph(false, false))
	expectedIgnores := NewSet(mkpkg("c"), Package("errors"))
	assertSetsEqual(t, deps.Ignored, expectedIgnores, "Ignored")
}

func mkpkg(rel string) Package {
	if rel == "" {
		return Package(basePkg)
	}
	return Package(basePkg + "/" + rel)
}

func expectedGraph(includeStdlib, includeTests bool) Graph {
	expected := NewGraph()
	for pkg, imports := range expectations {
		imps := expected.Pkg(mkpkg(pkg))
		for _, imp := range imports {
			imps.Insert(mkpkg(imp))
		}
		if includeStdlib && pkg != "" {
			imps.Insert(Package("errors"))
		}
		if includeTests && pkg != "" {
			imps.Insert(mkpkg("c"))
		}
	}
	if includeStdlib {
		expected.Pkg(Package("errors"))
	}
	if includeTests {
		imps := expected.Pkg(mkpkg("c"))
		if includeStdlib {
			imps.Insert(Package("errors"))
		}
	}
	return expected
}

func assertGraphsEqual(t *testing.T, actual, expected Graph) {
	// Check for missing packages.
	for pkg, imports := range expected {
		assertSetsEqual(t, actual[pkg], imports, string(pkg))
	}
	// Check for unexpected packages.
	for pkg := range actual {
		assert.Contains(t, expected, pkg, "Unexpected package %s", pkg)
	}
}

func assertSetsEqual(t *testing.T, actual, expected Set, ctx string) {
	// Check for missing items.
	for k := range expected {
		assert.Contains(t, actual, k, "[%s] Missing expected item %s", ctx, k)
	}
	// Check for unexpected items.
	for k := range actual {
		assert.Contains(t, expected, k, "[%s] Unexpected item %s", ctx, k)
	}
}

func TestStripVendor(t *testing.T) {
	tests := []struct{ path, expected string }{
		{"github.com/google/godepq/vendor/github.com/google/cadvisor/manager", "github.com/google/cadvisor/manager"},
		{"/vendor/github.com/google/cadvisor/manager", "github.com/google/cadvisor/manager"},
		{"github.com/google/cadvisor/manager", "github.com/google/cadvisor/manager"},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, stripVendor(test.path), "stripVendor(%s)", test.path)
	}
}
