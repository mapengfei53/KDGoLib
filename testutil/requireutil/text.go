package requireutil

import (
	"testing"

	"github.com/pmezard/go-difflib/difflib"
)

// RequireText if expected != actual, show error of diff detail
func RequireText(t *testing.T, expected string, actual string) {
	if expected == actual {
		return
	}

	diff := difflib.ContextDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
		Eol:      "\n",
	}
	result, _ := difflib.GetContextDiffString(diff)
	t.Fatal(result)
}
