package fancyfmt

import (
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/dave/dst"
)

const (
	widthLimit = 16
)

func formatMultiline(file *dst.File) error {
	dst.Inspect(file, func(node dst.Node) bool {
		switch v := node.(type) {
		case *dst.FuncDecl:
			multilineFuncDeclParams(v)
			multilineFuncDeclResults(v)
		case *dst.CallExpr:
			var isMultiline bool
			for _, p := range v.Args {
				if p.Decorations().Before == dst.NewLine {
					isMultiline = true
					break
				}
			}
			if !isMultiline {
				return true
			}
			for _, p := range v.Args {
				p.Decorations().Before = dst.NewLine
				p.Decorations().After = dst.NewLine
			}
		case *dst.CompositeLit:
			var isMultiline bool
			for _, v := range v.Elts {
				if v.Decorations().Before == dst.NewLine {
					isMultiline = true
					break
				}
			}
			if !isMultiline {
				return true
			}
			if possibleFormatting(v) {
				return true
			}
			for _, p := range v.Elts {
				p.Decorations().Before = dst.NewLine
				p.Decorations().After = dst.NewLine
			}
		case *dst.SelectorExpr:
			multilineSelector(v)
		}

		return true
	})

	return nil
}

func multilineFuncDeclParams(v *dst.FuncDecl) {
	var isMultiline bool
	for _, p := range v.Type.Params.List {
		if p.Decs.Before == dst.NewLine || p.Decs.After == dst.NewLine {
			isMultiline = true
			break
		}
	}
	if !isMultiline {
		return
	}
	for _, p := range v.Type.Params.List {
		p.Decorations().Before = dst.NewLine
		p.Decorations().After = dst.NewLine
	}
	return
}

func multilineFuncDeclResults(v *dst.FuncDecl) {
	var isMultiline bool
	if v.Type.Results == nil {
		return
	}
	for _, p := range v.Type.Results.List {
		if p.Decs.Before == dst.NewLine || p.Decs.After == dst.NewLine {
			isMultiline = true
			break
		}
	}
	if !isMultiline {
		return
	}
	for _, p := range v.Type.Results.List {
		p.Decorations().Before = dst.NewLine
		p.Decorations().After = dst.NewLine
	}
	return
}

func possibleFormatting(l *dst.CompositeLit) bool {
	if len(l.Elts) == 0 {
		return false
	}

	if l.Type == nil {
		return false
	}

	// only arrays are supported
	v, ok := l.Type.(*dst.ArrayType)
	if !ok {
		return false
	}

	if ensureFormat(l) {
		return true
	}

	// exit if an array type is not one of integer number types
	id, ok := v.Elt.(*dst.Ident)
	if !ok {
		return false
	}
	var width int
	switch id.Name {
	case "byte", "int8", "uint8":
		width = 2
	case "int", "int16", "int32", "int64", "uint", "uint16", "uint32", "uint64":
	default:
		return false
	}

	// checks if all elements are actual integers: expressions, constants, etc are not allowed, only right integer
	// numbers
	for _, e := range l.Elts {
		// don't fix anything if there's a comment
		if len(e.Decorations().Start) > 0 || len(e.Decorations().End) > 0 {
			return false
		}
		switch v := e.(type) {
		case *dst.BasicLit:
			if v.Kind != token.INT {
				return false
			}
			if strings.HasPrefix(v.Value, "0") && v.Value != "0" && !strings.HasPrefix(v.Value, "0x") {
				// уже используется бинарное либо 8-ричное предтавление чиселек, такое не трогаем, оно наверняка не
				// просто так
				return false
			}
		default:
			return false
		}
	}

	// ensure formatting
	sq := int(math.Floor(math.Sqrt(float64(len(l.Elts))) + 0.9999))
	if sq > widthLimit {
		sq = widthLimit
	}

	// tries to fit all numbers into a square up to 16 elements in width and height.
	for i, el := range l.Elts {
		if i%sq == 0 {
			el.Decorations().Before = dst.NewLine
		} else {
			el.Decorations().Before = dst.None
		}

		if i == len(l.Elts)-1 {
			el.Decorations().After = dst.NewLine
		} else {
			el.Decorations().After = dst.None
		}
		if width == 0 {
			continue
		}

		// replace byte number into its hex representation
		v := el.(*dst.BasicLit)
		var value uint64
		if strings.HasPrefix(v.Value, "0x") {
			value, _ = strconv.ParseUint(v.Value[2:], 16, 64)
		} else {
			value, _ = strconv.ParseUint(v.Value, 10, 64)
		}
		resval := strconv.FormatUint(value, 16)
		if len(resval) < width {
			resval = strings.Repeat("0", width-len(resval)) + resval
		}
		v.Value = "0x" + resval
	}

	return true
}

func ensureFormat(l *dst.CompositeLit) bool {
	if l.Elts[0].Decorations().Before != dst.NewLine {
		return false
	}
	var lineWidth int
	lineCount := 1
	widthCount := 1
	for _, e := range l.Elts {
		if len(e.Decorations().Start) > 0 || len(e.Decorations().End) > 0 {
			// found a comment. Don't put any formatting, just a new lines before the first and after the last element
			l.Elts[0].Decorations().Before = dst.NewLine
			l.Elts[len(l.Elts)-1].Decorations().After = dst.NewLine
			return true
		}
		if lineCount > 1 {
			continue
		}
		if e.Decorations().After == dst.NewLine {
			lineWidth = widthCount
			lineCount++
		} else {
			widthCount++
		}
	}

	if lineCount == 1 {
		// строка только одна, нас такое не интересует
		return false
	}

	// the first line setups array width, make every line to have the same length except, most probably, the last one
	for i, el := range l.Elts {
		if i%lineWidth == 0 {
			el.Decorations().Before = dst.NewLine
		} else {
			el.Decorations().Before = dst.None
		}

		if i == len(l.Elts)-1 {
			el.Decorations().After = dst.NewLine
		} else {
			el.Decorations().After = dst.None
		}
	}

	return true
}

func multilineSelector(x *dst.SelectorExpr) bool {
	var hasNewLine bool
	switch v := x.X.(type) {
	case *dst.CallExpr:
		hasNewLine = multlineSelectorFunc(v)
	case *dst.SelectorExpr:
		hasNewLine = multilineSelector(v)
	}
	if x.Sel.Decorations().Before == dst.NewLine {
		return true
	}
	if hasNewLine {
		x.Sel.Decorations().Before = dst.NewLine
		return true
	}

	return false
}

func multlineSelectorFunc(x *dst.CallExpr) bool {
	switch v := x.Fun.(type) {
	case *dst.CallExpr:
		return multlineSelectorFunc(v)
	case *dst.SelectorExpr:
		return multilineSelector(v)
	}

	return false
}
