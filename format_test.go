package fancyfmt_test

import (
	"go/parser"
	"go/token"
	"io"
	"os"
	"testing"

	"github.com/sirkon/errors"
	"github.com/sirkon/fancyfmt"
)

func TestFormat(t *testing.T) {
	grouper, err := fancyfmt.DefaultImportsGrouper()
	if err != nil {
		t.Fatal(errors.Wrap(err, "get default imports grouper"))
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(
		fset,
		"testdata/sample",
		nil,
		parser.AllErrors|parser.ParseComments,
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "parse testdata/sample"))
	}

	res, err := fancyfmt.Format(fset, file, grouper)
	if err != nil {
		t.Fatal(errors.Wrap(err, "format testdata/sample"))
	}

	if _, err := io.Copy(os.Stdout, res); err != nil {
		t.Fatal(errors.Wrap(err, "output formatted data"))
	}
}
