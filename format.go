package fancyfmt

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"sort"
	"strconv"

	"github.com/sirkon/dst"
	"github.com/sirkon/dst/decorator"
	"github.com/sirkon/errors"
	"golang.org/x/tools/go/ast/astutil"
)

// Format formats given AST tree
func Format(fset *token.FileSet, file *ast.File, content []byte, grouper ImportsGrouper) (io.Reader, error) {
	// work with imports
	imports, cComment := clearImports(fset, file)
	_, dfile, err := addImports(fset, file, grouper, imports, cComment)
	if err != nil {
		return nil, errors.Wrap(err, "rebuild import statements")
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

type pkgPath = string
type pkgAlias = string

// clearImports stores all imports into returning map and remove them from the file
func clearImports(fset *token.FileSet, file *ast.File) (_ map[pkgPath]pkgAlias, cComments *ast.CommentGroup) {
	imports := map[pkgPath]pkgAlias{}
	var importStart *token.Pos
	var importEnd *token.Pos
	var importCPos *token.Pos
	ast.Inspect(file, func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.GenDecl:
			if v.Tok != token.IMPORT {
				return true
			}

			if importStart == nil {
				pos := v.Pos()
				importStart = &pos
			}
			pos := v.End()
			importEnd = &pos

		case *ast.ImportSpec:
			if v.Name != nil {
				imports[unqoute(v.Path.Value)] = v.Name.Name
			} else {
				imports[unqoute(v.Path.Value)] = ""
			}
			if v.Path.Value == `"C"` {
				tmp := v.Pos()
				importCPos = &tmp
			}
			if v.Doc != nil && len(v.Doc.List) > 0 {
			comment:
				for {
					for i, f := range file.Comments {
						for _, d := range v.Doc.List {
							if f.Pos() == d.Pos() {
								file.Comments = append(file.Comments[:i], file.Comments[i+1:]...)
								continue comment
							}
						}
					}
					break
				}
			}
		}

		return true
	})

	// remove imports
	for imp, alias := range imports {
		if alias != "" {
			astutil.DeleteNamedImport(fset, file, alias, imp)
		} else {
			astutil.DeleteImport(fset, file, imp)
		}
	}

	// remove comments within imports
	if importStart != nil {
		var cmts []*ast.CommentGroup
		if importCPos != nil {
			cLexicPos := fset.Position(*importCPos)
		cImportsCommentsLoop:
			for _, cmt := range file.Comments {
				commentEndLexicPos := fset.Position(cmt.End()).Line
				switch {
				case commentEndLexicPos+1 == cLexicPos.Line:
					cComments = cmt
					break cImportsCommentsLoop
				case commentEndLexicPos > cLexicPos.Line:
					break cImportsCommentsLoop
				}
			}
		}
		for _, cmt := range file.Comments {
			if *importStart <= cmt.Pos() && cmt.End() <= *importEnd || fset.Position(cmt.End()).Line+1 == fset.Position(*importStart).Line {
				continue
			}
			cmts = append(cmts, cmt)
		}
		file.Comments = cmts
	}

	return imports, cComments
}

func unqoute(v string) string {
	res, _ := strconv.Unquote(v)
	return res
}

type imp struct {
	alias string
	path  string
}

// addImports adds imports removed just before and setups proper line distancing within import statments. Returns dst
// file as all further work needed it
func addImports(
	fset *token.FileSet,
	file *ast.File,
	grouper ImportsGrouper,
	imports map[pkgPath]pkgAlias,
	cComment *ast.CommentGroup,
) (*token.FileSet, *dst.File, error) {
	// group imports
	groups := map[int][]imp{}
	for path, alias := range imports {
		weight := grouper.Weight(path)
		groups[weight] = append(groups[weight], imp{
			alias: alias,
			path:  path,
		})
	}

	// sort each group
	var weights []int
	for weight, imps := range groups {
		weights = append(weights, weight)
		sort.Slice(imps, func(i, j int) bool {
			return imps[i].path < imps[j].path
		})
	}

	// insert imports
	sort.Ints(weights)
	for _, weight := range weights {
		for _, imp := range groups[weight] {
			imp := imp

			// someAdded = true
			if imp.alias != "" {
				astutil.AddNamedImport(fset, file, imp.alias, imp.path)
			} else {
				astutil.AddImport(fset, file, imp.path)
			}
			continue
		}
	}

	// need to reorder imports as astutil setups its own order which may be not what is needed for us

	// look for import which is not C
	var decl *ast.GenDecl
	_, cImportNotPassed := groups[importGroupC]
loop:
	for _, dec := range file.Decls {
		switch v := dec.(type) {
		case *ast.GenDecl:
			if v.Tok != token.IMPORT {
				continue
			}
			if cImportNotPassed {
				cImportNotPassed = false
				continue
			}
			decl = v
			break loop
		default:
			continue
		}
	}
	if decl != nil {
		// import statement that matters is in decl, set up a proper order
		reorderImports(weights, groups, decl)
	}

	// utter crap, format source code then parse it into dst
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return nil, nil, errors.Wrap(err, "format with new refreshed imports")
	}

	fset = token.NewFileSet()
	file, err := parser.ParseFile(fset, "", buf.Bytes(), parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse intermediate state")
	}

	dfile, err := decorator.DecorateFile(fset, file)
	if err != nil {
		return nil, nil, errors.Wrap(err, "convert to dst")
	}

	// intoduce proper line spacing
	_, cImportNotPassed = groups[importGroupC]
dloop:
	for _, dec := range dfile.Decls {
		switch v := dec.(type) {
		case *dst.GenDecl:
			if v.Tok != token.IMPORT {
				continue
			}
			if cComment != nil {
				var lines []string
				for _, cmt := range cComment.List {
					lines = append(lines, cmt.Text)
				}
				dec.Decorations().Start.Prepend(lines...)
			}
			dec.Decorations().End.Append("\n\n\n")
			if cImportNotPassed {
				cImportNotPassed = false
				// add an empty line after import "C"
				continue
			}

			// adds empty line before imports of each group except the first one
			var offset int
			for _, group := range weights {
				if group == importGroupC {
					continue
				}
				prevOff := offset
				offset += len(groups[group])
				if prevOff == 0 {
					continue
				}
				v.Specs[prevOff].Decorations().Start.Prepend("\n")
			}

			break dloop
		}
	}

	return fset, dfile, nil
}

func reorderImports(weights []int, groups map[int][]imp, decl *ast.GenDecl) {
	importOrder := map[pkgPath]int{}
	var i int
	for _, group := range weights {
		if group == importGroupC {
			continue
		}
		imports := groups[group]
		for _, imp := range imports {
			importOrder[imp.path] = i
			i++
		}
	}
	sort.Slice(decl.Specs, func(i, j int) bool {
		vi := decl.Specs[i].(*ast.ImportSpec)
		vj := decl.Specs[j].(*ast.ImportSpec)

		oi := importOrder[unqoute(vi.Path.Value)]
		oj := importOrder[unqoute(vj.Path.Value)]

		return oi < oj
	})
}
