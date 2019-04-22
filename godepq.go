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

	"github.com/google/godepq/deps"
)

var (
	// TODO: add support for multiple from / to packages
	from          = flag.String("from", "", "root package")
	to            = flag.String("to", "", "target package for querying dependency paths")
	toRegex       = flag.String("toregex", "", "target package regex for querying dependency paths")
	ignore        = flag.String("ignore", "", "regular expression for packages to ignore")
	include       = flag.String("include", "", "regular expression for packages to include (excluding packages matching -ignore)")
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

	fromPkg, baseDir, err := resolveSource(*from, wd)
	if err != nil {
		return err
	}
	var toPkg deps.Package
	if *to != "" {
		toPkg, err = deps.Resolve(*to, wd, build.Default)
		if err != nil {
			return err
		}
	}

	builder := deps.Builder{
		Roots:         []deps.Package{fromPkg},
		IncludeTests:  *includeTests,
		IncludeStdlib: *includeStdlib,
		BuildContext:  build.Default,
		BaseDir:       baseDir,
	}

	if *ignore != "" {
		ignoreRegexp, err := regexp.Compile(*ignore)
		if err != nil {
			return err
		}
		builder.Ignored = []*regexp.Regexp{ignoreRegexp}
	}

	if *include != "" {
		includeRegexp, err := regexp.Compile(*include)
		if err != nil {
			return err
		}
		builder.Included = []*regexp.Regexp{includeRegexp}
	}

	graph, err := builder.Build()
	if err != nil {
		return err
	}

	var result deps.Graph
	var endCond func(deps.Package) bool
	if toPkg != "" {
		endCond = func(pkg deps.Package) bool {
			return pkg == toPkg
		}
	} else if *toRegex != "" {
		r := regexp.MustCompile(*toRegex)
		endCond = func(pkg deps.Package) bool {
			return r.MatchString(string(pkg))
		}
	}

	if endCond != nil {
		if *allPaths {
			result = graph.Forward.AllPathsCond(fromPkg, endCond)
		} else {
			path := graph.Forward.SomePathCond(fromPkg, endCond)
			result = deps.NewGraph()
			result.AddPath(path)
		}
	} else {
		result = graph.Forward
	}

	if result == nil || len(result) == 0 {
		dst := string(toPkg)
		if *toRegex != "" {
			dst = *toRegex
		}
		fmt.Fprintf(os.Stderr, "No path found from %q to %q\n", fromPkg, dst)
		os.Exit(1)
	}

	switch *output {
	case "list":
		printList(fromPkg, result)
		return nil
	case "dot":
		printDot(fromPkg, result, graph.Info)
		return nil
	default:
		return fmt.Errorf("Unknown output format %q", *output)
	}
}

func validateFlags() error {
	if *from == "" {
		return errors.New("-from must be set")
	}

	if *allPaths && *to == "" && *toRegex == "" {
		return errors.New("-all-paths requires a -to package")
	}

	if *to != "" && *toRegex != "" {
		return errors.New("only one of -to and -toregex may be set")
	}

	if *toRegex != "" {
		if _, err := regexp.Compile(*toRegex); err != nil {
			return fmt.Errorf("invalid -toregex: %v", err)
		}
	}

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flag.Args())
	}

	if *ignore != "" && *ignore == *include {
		return errors.New("-include can not be the same as -ignore")
	}
	return nil
}

func printList(root deps.Package, paths deps.Graph) {
	fmt.Println("Packages:")
	for _, pkg := range paths.List(root) {
		fmt.Printf("  %s\n", pkg)
	}
}

func printDot(root deps.Package, paths deps.Graph, pkgInfo map[deps.Package]*deps.DependencyInfo) {
	labelFn := func(pkg deps.Package) string {
		return fmt.Sprintf("%s (%d)", pkg, pkgInfo[pkg].LOC)
	}
	fmt.Println(paths.Dot(root, labelFn))
}

// resolveSource resolves the import path, and determines the base directory to resolve future
// imports from.
// If the resolved import is vendored, then future imports should use the same vendored sources.
// Otherwise, future imports should be resolved with the source's vendor directory.
func resolveSource(importPath, workingDir string) (deps.Package, string, error) {
	pkg, err := build.Default.Import(importPath, workingDir, build.FindOnly)
	if err != nil {
		return "", "", fmt.Errorf("unable to resolve %q: %v", importPath, err)
	}
	src, vendored := deps.StripVendor(deps.Package(pkg.ImportPath))
	if vendored {
		return src, workingDir, nil
	}
	return src, pkg.Dir, nil
}
