package fancyfmt

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/sirkon/errors"
)

// TODO delete after dst will get type params support

// restoreTypeParameters restores type parameters deleted by dst library, which does not support type params
// as of now.
func restoreTypeParameters(fset *token.FileSet, orig *ast.File, origSrc []byte, formatted []byte) ([]byte, error) {
	now, err := parser.ParseFile(fset, "", formatted, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, errors.Wrap(err, "parse original file")
	}

	inserts := map[int][]byte{}

	ast.Inspect(orig, func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.TypeSpec:
			if v.TypeParams == nil {
				return true
			}

			var vv *ast.TypeSpec
			ast.Inspect(now, func(n ast.Node) bool {
				vvv, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				if v.Name.Name == vvv.Name.Name {
					vv = vvv
					return false
				}

				return true
			})

			start := fset.Position(v.TypeParams.Opening).Offset
			finish := fset.Position(v.TypeParams.Closing).Offset
			inserts[fset.Position(vv.Name.End()).Offset] = origSrc[start : finish+1]
		case *ast.FuncDecl:
			if v.Type.TypeParams == nil {
				return true
			}

			var vv *ast.FuncDecl
			ast.Inspect(now, func(n ast.Node) bool {
				vvv, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				if vvv.Name.Name == v.Name.Name {
					vv = vvv
					return false
				}

				return true
			})

			start := fset.Position(v.Type.TypeParams.Opening).Offset
			finish := fset.Position(v.Type.TypeParams.Closing).Offset
			inserts[fset.Position(vv.Name.End()).Offset] = origSrc[start : finish+1]
		}

		return true
	})

	res := make([]byte, 0, len(formatted)+1024)
	for i, b := range formatted {
		if v, ok := inserts[i]; ok {
			res = append(res, v...)
		}

		res = append(res, b)
	}

	return res, nil
}
