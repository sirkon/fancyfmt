package fancyfmt

import (
	"bytes"
	"go/ast"
	"go/token"
	"io"
	"sort"
	"strconv"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/sirkon/errors"
)

// Format formats given AST tree
func Format(fset *token.FileSet, file *ast.File, content []byte, grouper ImportsGrouper) (io.Reader, error) {
	dfile, err := decorator.DecorateFile(fset, file)
	if err != nil {
		return nil, errors.Wrap(err, "get ast decoration")
	}

	// Ищем первую ноду не import "C"
	impStart := -1
	impFinish := impStart
	var imports []dst.Spec
	for i, decl := range dfile.Decls {
		g, ok := decl.(*dst.GenDecl)
		if !ok {
			break
		}

		if g.Tok != token.IMPORT {
			break
		}

		impSpec := g.Specs[0].(*dst.ImportSpec)
		if impSpec.Path.Value == `"C"` {
			continue
		}

		if impStart == -1 {
			impStart = i
			impFinish = i
		} else {
			impFinish = i
		}

		imports = append(imports, g.Specs...)
	}
	if impStart >= 0 {
		dfile.Decls = append(dfile.Decls[:impStart], dfile.Decls[impFinish+1:]...)
	}
	weights := map[string]int{}
	sort.Slice(imports, func(i, j int) bool {
		di := imports[i].(*dst.ImportSpec)
		dj := imports[j].(*dst.ImportSpec)
		pi := unqoute(di.Path.Value)
		wi := grouper.Weight(pi)
		weights[di.Path.Value] = wi
		pj := unqoute(dj.Path.Value)
		wj := grouper.Weight(pj)
		weights[dj.Path.Value] = wj
		if wi != wj {
			return wi < wj
		}
		return pi < pj
	})
	for i, spec := range imports {
		if i == 0 {
			continue
		}

		fi := imports[i-1].(*dst.ImportSpec).Path.Value
		si := spec.(*dst.ImportSpec).Path.Value
		if weights[fi] == weights[si] {
			spec.(*dst.ImportSpec).Decorations().Before = dst.None
			spec.(*dst.ImportSpec).Decorations().After = dst.NewLine
		} else {
			spec.(*dst.ImportSpec).Decorations().Before = dst.EmptyLine
			spec.(*dst.ImportSpec).Decorations().After = dst.NewLine
		}
	}
	if impStart >= 0 {
		decls := make([]dst.Decl, 0, len(dfile.Decls)+1)
		decls = append(decls, dfile.Decls[:impStart]...)
		decls = append(decls, &dst.GenDecl{
			Tok:    token.IMPORT,
			Lparen: true,
			Specs:  imports,
			Rparen: true,
		})
		decls = append(decls, dfile.Decls[impStart:]...)

		dfile.Decls = decls
	}
	if err := formatMultiline(dfile); err != nil {
		return nil, errors.Wrap(err, "set up multiline formatting")
	}

	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, dfile); err != nil {
		return nil, errors.Wrap(err, "format result")
	}

	return &buf, nil
}

func unqoute(v string) string {
	res, _ := strconv.Unquote(v)
	return res
}
