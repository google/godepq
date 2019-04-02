/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

// This package sets up an import DAG for testdata purposes.
// The structure looks like:
//        a          b         c
//      /   \        |
//     aa   ab       ba
//     |
//    aaa
//
// With every test depending on c, and every non-test depending on "errors".
package testdata
