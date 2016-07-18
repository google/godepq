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
	from          = flag.String("from", "", "root package")
	to            = flag.String("to", "", "target package for querying dependency paths")
	ignore        = flag.String("ignore", "", "regular expression for packages to ignore")
	includeTests  = flag.Bool("include-tests", false, "whether to include test imports")
	includeStdlib = flag.Bool("include-stdlib", false, "whether to include go standard library imports")
	allPaths      = flag.Bool("all-paths", false, "whether to include all paths in the result")
	output        = flag.String("o", "list", "{list: print path(s), dot: export dot graph}")
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	err := validateFlags()
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	fromPkg, err := Resolve(*from, wd, build.Default)
	if err != nil {
		return err
	}
	toPkg, err := Resolve(*to, wd, build.Default)
	if err != nil {
		return err
	}

	builder := Builder{
		Roots:         []Package{Package(fromPkg)},
		IncludeTests:  *includeTests,
		IncludeStdlib: *includeStdlib,
		BuildContext:  build.Default,
		BaseDir:       wd,
	}

	if *ignore != "" {
		ignoreRegexp, err := regexp.Compile(*ignore)
		if err != nil {
			return err
		}
		builder.Ignored = []*regexp.Regexp{ignoreRegexp}
	}

	deps, err := builder.Build()
	if err != nil {
		return err
	}

	var result Graph
	if toPkg != "" {
		if *allPaths {
			result = deps.Forward.AllPaths(Package(fromPkg), Package(toPkg))
		} else {
			path := deps.Forward.SomePath(Package(fromPkg), Package(toPkg))
			result = NewGraph()
			result.AddPath(path)
		}
	} else {
		result = deps.Forward
	}

	if result == nil || len(result) == 0 {
		fmt.Printf("No path found from %q to %q\n", fromPkg, toPkg)
		os.Exit(1)
	}

	switch *output {
	case "list":
		printList(Package(fromPkg), result)
		return nil
	case "dot":
		printDot(Package(fromPkg), result)
		return nil
	default:
		return fmt.Errorf("Unknown output format %q", *output)
	}
}

func validateFlags() error {
	if *from == "" {
		return errors.New("-from must be set")
	}

	if *allPaths && *to == "" {
		return errors.New("-all-paths requires a -to package")
	}

	if len(flag.Args()) != 0 {
		return fmt.Errorf("Unexpected positional arguments: %v", flag.Args())
	}

	return nil
}

func printList(root Package, paths Graph) {
	fmt.Println("Packages:")
	for _, pkg := range paths.List(root) {
		fmt.Printf("  %s\n", pkg)
	}
}

func printDot(root Package, paths Graph) {
	fmt.Println(paths.Dot(root))
}
