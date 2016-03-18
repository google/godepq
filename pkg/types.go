/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

package pkg

type Package string

const NullPackage Package = ""

type Path []Package

func (p Path) Last() Package {
	return p[len(p)-1]
}

func (p Path) Pop() Path {
	return p[:len(p)-1]
}

type present struct{}

type Set map[Package]present

func NewSet(pkgs ...Package) Set {
	set := make(Set, len(pkgs))
	for _, pkg := range pkgs {
		set[pkg] = present{}
	}
	return set
}

func (ps Set) Insert(pkg Package) {
	ps[pkg] = present{}
}

func (ps Set) Delete(pkg Package) {
	delete(ps, pkg)
}

func (ps Set) Has(pkg Package) bool {
	_, found := ps[pkg]
	return found
}
