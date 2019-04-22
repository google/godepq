/*
Copyright (c) 2013-2016 the Godepq Authors

Use of this source code is governed by a MIT-style
license that can be found in the LICENSE file or at
https://opensource.org/licenses/MIT.
*/

package deps

import (
	"fmt"
	"os"

	"github.com/hhatto/gocloc"
)

type LinesOfCode map[string]int32

func NewLinesOfCode() LinesOfCode {
	return make(LinesOfCode)
}

// SetLinesOfCode analzes the source code using gcloc
// and sets the lines for code for the pa
func (l LinesOfCode) SetLinesOfCode(path string) {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	processor := gocloc.NewProcessor(languages, options)
	result, err := processor.Analyze([]string{path})
	if err != nil {
		fmt.Printf("gocloc failed to analyze. error: %v\n", err)
		os.Exit(1)
		return
	}

	l[path] = result.Total.Code
}
