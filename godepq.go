/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"go/build"
	"os"
	"regexp"

	. "github.com/google/godepq/pkg"
)

var (
	// TODO: add support for multiple from / to packages
	fromPkg       = flag.String("from", "", "root package")
	toPkg         = flag.String("to", "", "target package for querying dependency paths")
	ignore        = flag.String("ignore", "", "regular expression for packages to ignore")
	includeTests  = flag.Bool("include-tests", false, "whether to include test imports")
	includeStdlib = flag.Bool("include-stdlib", false, "whether to include go standard library imports")
	allPaths      = flag.Bool("all-paths", false, "whether to include all paths in the result")
	output        = flag.String("o", "list", "{list: print path(s), dot: export dot graph}")
)

func main() {
	flag.Parse()
	validateFlags()

	builder := Builder{
		Roots:         []Package{Package(*fromPkg)},
		IncludeTests:  *includeTests,
		IncludeStdlib: *includeStdlib,
		BuildContext:  build.Default,
	}
	baseDir, err := os.Getwd()
	handleError(err)
	builder.BaseDir = baseDir

	if *ignore != "" {
		ignoreRegexp, err := regexp.Compile(*ignore)
		handleError(err)
		builder.Ignored = []*regexp.Regexp{ignoreRegexp}
	}

	deps, err := builder.Build()
	handleError(err)

	var result Graph
	if *toPkg != "" {
		if *allPaths {
			result = deps.Forward.AllPaths(Package(*fromPkg), Package(*toPkg))
		} else {
			path := deps.Forward.SomePath(Package(*fromPkg), Package(*toPkg))
			result = NewGraph()
			result.AddPath(path)
		}
	} else {
		result = deps.Forward
	}

	if result == nil || len(result) == 0 {
		fmt.Printf("No path found from %q to %q\n", *fromPkg, *toPkg)
		os.Exit(1)
	}

	switch *output {
	case "list":
		printList(Package(*fromPkg), result)
	case "dot":
		printDot(Package(*fromPkg), result)
	default:
		handleError(fmt.Errorf("Unknown output format %q", *output))
	}
}

func validateFlags() {
	if *fromPkg == "" {
		handleError(errors.New("-from must be set"))
	}

	if *allPaths && *toPkg == "" {
		handleError(errors.New("-all-paths requires a -to package"))
	}

	if len(flag.Args()) != 0 {
		handleError(fmt.Errorf("Unexpected positional arguments: %v", flag.Args()))
	}
}

func printList(root Package, paths Graph) {
	fmt.Println("Packages:")
	paths.DepthLast(root, func(pkg Package, _ Set, _ Path) bool {
		fmt.Printf("  %s\n", pkg)
		return true
	})
}

func printDot(root Package, paths Graph) {
	fmt.Println("digraph godeps {")

	nextId := 0
	ids := make(map[Package]int, len(paths))
	getId := func(pkg Package) int {
		if id, ok := ids[pkg]; ok {
			return id
		}
		ids[pkg] = nextId
		nextId++
		return nextId - 1
	}

	paths.DepthFirst(root, func(pkg Package, edges Set, _ Path) bool {
		pkgId := getId(pkg)
		fmt.Printf("%d [label=\"%s\"];\n", pkgId, pkg)
		for edge := range edges {
			fmt.Printf("%d -> %d;\n", pkgId, getId(edge))
		}
		return true
	})

	fmt.Println("}")
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
