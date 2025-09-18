package fancyfmt

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirkon/errors"
	"github.com/sirkon/jsonexec"
	"golang.org/x/tools/go/packages"
)

const (
	ImportGroupC = iota
	ImportGroupStd
	ImportGroup3rdParty
	ImportGroupCurrent
	ImportGroupRelative
)

type defaultImportGrouper string

// Weight to implement ImportsGrouper
func (g defaultImportGrouper) Weight(path string) int {
	switch {
	case path == "C":
		return ImportGroupC
	case isStdlibPackage(path):
		return ImportGroupStd
	case g != "" && g.isSubPkg(path):
		return ImportGroupCurrent
	case strings.HasPrefix(path, "."):
		return ImportGroupRelative
	default:
		return ImportGroup3rdParty
	}
}

func (g defaultImportGrouper) isSubPkg(pkg string) bool {
	if pkg == string(g) {
		return true
	}
	ok, err := path.Match(path.Join(string(g), "*"), pkg)
	if err == nil && ok {
		return true
	}

	return false
}

// DefaultImportsGrouper provides an import grouper with a policy that is supposed to be the default:
//
//	"C" - 0
//	Standard library - 1
//	3rd party - 2
//	Current project - 3
//	Relative imports - 4
//
// It tried to determine a current project once called and may return an error if it failed to detect it. Use
// DefaultImportGroupsWithCurrent if you don't need it or need to set up your own
func DefaultImportsGrouper() (ImportsGrouper, error) {
	curdir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "get current directory")
	}

	// try to find project path in its go.mod file
	curproject, err := findCurrentProjectInGoMod(curdir)
	if err != nil {
		return nil, errors.Wrap(err, "look for go.mod file in "+curproject)
	}
	if curproject == "" {
		return nil, errors.New("empty module path")
	}

	oncer.Do(initStdlibPackages)
	return defaultImportGrouper(curproject), nil
}

// DefaultImportGroupsWithCurrent the same s DefaultImportsGroups just no current package set
func DefaultImportGroupsWithCurrent(current string) ImportsGrouper {
	oncer.Do(initStdlibPackages)
	return defaultImportGrouper(current)
}

func findCurrentProjectInGoMod(curdir string) (string, error) {
	var data struct {
		Module struct {
			Path string
		}
	}
	if err := jsonexec.Run(&data, "go", "mod", "edit", "--json"); err != nil {
		return "", errors.Wrap(err, "get module info")
	}

	return data.Module.Path, nil
}

var oncer sync.Once
var stdlibPackages map[string]struct{}

const stdPkgsCache = "fancy-fmt-std-packages-cache"

func initStdlibPackages() {
	stdlibPackages = map[string]struct{}{}

	cacheFilePath := filepath.Join(os.TempDir(), stdPkgsCache)
	data, err := ioutil.ReadFile(cacheFilePath)
	var pkgs []string
	var noCache bool
	if err != nil {
		noCache = true
		pks, err := packages.Load(&packages.Config{Mode: packages.NeedName}, "std")
		if err != nil {
			panic(err)
		}
		for _, p := range pks {
			pkgs = append(pkgs, p.PkgPath)
		}
	} else {
		pkgs = strings.Split(string(data), "\n")
	}

	for _, p := range pkgs {
		stdlibPackages[p] = struct{}{}
	}

	if noCache {
		data := strings.Join(pkgs, "\n")
		_ = ioutil.WriteFile(cacheFilePath, []byte(data), 0644)
	}
}

func isStdlibPackage(path string) bool {
	_, ok := stdlibPackages[path]
	return ok
}
