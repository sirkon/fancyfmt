package main

import (
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/sirkon/errors"
	"github.com/sirkon/message"

	"github.com/sirkon/fancyfmt"
)

func main() {
	var cli struct {
		Write          bool   `short:"w" help:"Write formatted files."`
		Recursive      bool   `short:"r" help:"Process directories recursively. This options requires -w|--write option to be enabled."`
		CurrentProject string `short:"c" help:"Use this value as the current project path"`

		Paths []string `arg:"" type:"path" help:"Paths to process. May be file or directory if recursive option is enabled, use '-'' to format stdin input."`
	}

	ctx := kong.Parse(&cli)
	ctx.Model.Name = "fancyfmt"
	if cli.Recursive && !cli.Write {
		ctx.Fatalf("recursive options requires write option on")
	}
	if len(cli.Paths) > 1 && !cli.Write {
		ctx.Fatalf("can only process the single path with write option set off, got %d path items", len(cli.Paths))
	}

	var importsGrouper fancyfmt.ImportsGrouper
	if cli.CurrentProject != "" {
		importsGrouper = fancyfmt.DefaultImportGroupsWithCurrent(cli.CurrentProject)
	} else {
		var err error
		importsGrouper, err = fancyfmt.DefaultImportsGrouper()
		if err != nil {
			message.Fatal("get imports grouper")
		}
	}

	for _, path := range cli.Paths {
		if filepath.Base(path) == "-" && len(cli.Paths) != 1 {
			message.Fatal("cannot combine stdin input with files or another stdin inputs")
		}
	}
	for _, p := range cli.Paths {
		if filepath.Base(p) == "-" {
			if err := processStdin(); err != nil {
				message.Fatal(err)
			}
			return
		}
		if err := process(p, cli.Recursive, cli.Write, importsGrouper); err != nil {
			message.Fatal(errors.Wrap(err, "process "+p))
		}
	}
}

func process(path string, recursive bool, write bool, grouper fancyfmt.ImportsGrouper) error {
	var paths []string
	stat, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, "check input path")
	}

	if stat.IsDir() {
		if !recursive {
			return errors.New("cannot process directory without recursive enabled")
		}
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				_, base := filepath.Split(path)
				if strings.HasPrefix(base, ".") && base != "." && base != ".." {
					return filepath.SkipDir
				}

				return nil
			}

			if strings.HasSuffix(path, ".go") {
				paths = append(paths, path)
			}

			return nil
		})
		if err != nil {
			return errors.Wrap(err, "walk directory")
		}
	} else {
		paths = append(paths, path)
	}

	for _, path := range paths {
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "read file content")
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, fileContent, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return errors.Wrap(err, "parse "+path)
		}

		res, err := fancyfmt.Format(fset, file, fileContent, grouper)
		if err != nil {
			return errors.Wrap(err, "format "+path)
		}

		if write {
			dir, base := filepath.Split(path)
			tmpFile, err := ioutil.TempFile(dir, base)
			if err != nil {
				return errors.Wrap(err, "create temporary file to save formatted data")
			}
			if _, err := io.Copy(tmpFile, res); err != nil {
				return errors.Wrap(err, "write formatted data into temporary file")
			}
			if err := os.Rename(tmpFile.Name(), path); err != nil {
				return errors.Wrap(err, "replace original source code with formatted one from "+tmpFile.Name())
			}
		} else {
			if _, err := io.Copy(os.Stdout, res); err != nil {
				return errors.Wrap(err, "copy to stdout")
			}
		}
	}

	return nil
}

func processStdin() error {
	input, err := io.ReadAll(os.Stdin)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "-", input, parser.AllErrors|parser.ParseComments)
	if err != nil {
		// do not annotate parsing error as it may have
		// error position info at the start
		message.Error("failed to parse stdin data")
		return err
	}

	grouper, err := fancyfmt.DefaultImportsGrouper()
	if err != nil {
		return errors.Wrap(err, "setup imports grouper")
	}
	res, err := fancyfmt.Format(fset, file, input, grouper)
	if err != nil {
		// same here, do not annotate formatting error
		message.Error("failed to format stdin data")
		return err
	}

	_, err = io.Copy(os.Stdout, res)
	if err != nil {
		return errors.Wrap(err, "write formatted source code into the stdout")
	}

	return nil
}
