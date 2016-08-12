/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

package pkg

import (
	"errors"
	"fmt"
	"go/build"
	"log"
	"regexp"
)

type Dependencies struct {
	// Map of package -> dependencies.
	Forward Graph
	// Packages which were ignored.
	Ignored Set
}

type Condition func(Dependencies) bool

// Resolve resolves import paths to a canonical, absolute form.
// Relative paths are resolved relative to basePath.
// It does not verify that the import is valid.
func Resolve(importPath, basePath string, bctx build.Context) (string, error) {
	pkg, err := bctx.Import(importPath, basePath, build.FindOnly)
	if err != nil {
		return "", fmt.Errorf("unable to resolve %q: %v", importPath, err)
	}
	return pkg.ImportPath, nil
}

type Builder struct {
	// The base directory for relative imports.
	BaseDir string
	// The roots of the dependency graph (source packages).
	Roots []Package
	// Stop building the graph if ANY conditions are met.
	TerminationConditions []Condition
	// Ignore any packages that match any of these patterns.
	// Tested on the resolved package path.
	Ignored []*regexp.Regexp
	// Include only packages that match any of these patterns.
	// Tested on the resolved package path.
	Included []*regexp.Regexp
	// Whether tests should be included in the dependencies.
	IncludeTests bool
	// Whether to include standard library packages
	IncludeStdlib bool
	// The build context for processing imports.
	BuildContext build.Context

	// Internal
	deps Dependencies
}

// Packages which should always be ignored.
var pkgBlacklist = NewSet(
	"C", // c imports, causes problems
)

func (b *Builder) Build() (Dependencies, error) {
	b.deps = Dependencies{
		Forward: NewGraph(),
		Ignored: NewSet(),
	}

	err := b.addAllPackages(b.Roots)
	if err == termination {
		err = nil // Ignore termination condition.
	}
	return b.deps, err
}

func (b *Builder) addAllPackages(pkgs []Package) error {
	for _, pkg := range pkgs {
		if !b.isAccepted(pkg) {
			log.Printf("Warning: ignoring root package %q", pkg)
			b.deps.Ignored.Insert(pkg)
			continue
		}

		// TODO: add support for recursive sub-packages.
		if err := b.addPackage(pkg); err != nil {
			return err
		}
	}
	return nil
}

var termination = errors.New("termination condition met")

// Recursively adds a package to the accumulated dependency graph.
func (b *Builder) addPackage(pkgName Package) error {
	pkg, err := b.BuildContext.Import(string(pkgName), b.BaseDir, 0)
	if err != nil {
		return err
	}

	pkgFullName := Package(pkg.ImportPath)
	if !b.isAccepted(pkgFullName) {
		b.deps.Ignored.Insert(pkgFullName)
		return nil
	}

	// Insert the package.
	b.deps.Forward.Pkg(pkgFullName)

	for _, condition := range b.TerminationConditions {
		if condition(b.deps) {
			return termination
		}
	}

	for _, imp := range b.getImports(pkg) {
		if !b.isAccepted(imp) {
			b.deps.Ignored.Insert(imp)
			continue
		}

		impPkg, err := b.BuildContext.Import(string(imp), b.BaseDir, 0)
		if err != nil {
			return err
		}

		isImpStdlib := impPkg.Goroot
		if isImpStdlib && !b.IncludeStdlib {
			continue
		}

		b.deps.Forward.Pkg(pkgFullName).Insert(imp)

		// If imp has not been added, add it now.
		if !b.deps.Forward.Has(imp) {
			if err := b.addPackage(imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Builder) getImports(pkg *build.Package) []Package {
	allImports := pkg.Imports
	if b.IncludeTests {
		allImports = append(allImports, pkg.TestImports...)
		allImports = append(allImports, pkg.XTestImports...)
	}
	var imports []Package
	found := make(map[string]struct{})
	for _, imp := range allImports {
		if imp == pkg.ImportPath {
			// Don't draw a self-reference when foo_test depends on foo.
			continue
		}
		if _, ok := found[imp]; ok {
			continue
		}
		found[imp] = struct{}{}
		imports = append(imports, Package(imp))
	}
	return imports
}

func (b *Builder) isIgnored(pkg Package) bool {
	if pkgBlacklist.Has(pkg) {
		return true
	}
	for _, r := range b.Ignored {
		if r.MatchString(string(pkg)) {
			return true
		}
	}
	return false
}

func (b *Builder) isIncluded(pkg Package) bool {
	if len(b.Included) == 0 {
		return true
	}
	for _, r := range b.Included {
		if r.MatchString(string(pkg)) {
			return true
		}
	}
	return false
}

// Detects if package name matches search criterias  
func (b *Builder) isAccepted(pkg Package) bool {
	return !b.isIgnored(pkg) && b.isIncluded(pkg)
}
