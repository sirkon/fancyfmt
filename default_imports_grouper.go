package fancyfmt

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirkon/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

const (
	importGroupC = iota
	importGroupStd
	importGroup3rdParty
	importGroupCurrent
	importGroupRelative
)

type defaultImportGrouper string

// Weight to implement ImportsGrouper
func (g defaultImportGrouper) Weight(path string) int {
	switch {
	case path == "C":
		return importGroupC
	case isStdlibPackage(path):
		return importGroupStd
	case g != "" && g.isSubPkg(path):
		return importGroupCurrent
	case strings.HasPrefix(path, "."):
		return importGroupRelative
	default:
		return importGroup3rdParty
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
//   "C" - 0
//   Standard library - 1
//   3rd party - 2
//   Current project - 3
//   Relative imports - 4
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
		curproject, err = deriveProjectPathFromGoPath()
		if err != nil {
			return nil, errors.Wrap(err, "derive package path as it would be a $GOPATH/src subdir")
		}
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
	for {
		gomodName := filepath.Join(curdir, "go.mod")
		gomodData, err := ioutil.ReadFile(gomodName)
		if err == nil {
			gomod, err := modfile.Parse(gomodName, gomodData, nil)
			if err != nil {
				return "", errors.Wrap(err, "parse "+gomodName)
			}

			return gomod.Module.Mod.Path, nil

		} else if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "open go.mod in "+curdir)
		}

		if curdir == "" || curdir == string(os.PathSeparator) {
			break
		}
		curdir, _ = filepath.Split(curdir)
	}

	return "", nil
}

func deriveProjectPathFromGoPath() (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "find current user home dir to use $HOME/go as $GOPATH")
		}

		gopath = filepath.Join(home, "go")
	}

	gosrc := filepath.Join(gopath, "src")
	curdir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "get user home dir")
	}
	rel, err := filepath.Rel(gosrc, curdir)
	if err != nil {
		return "", errors.Wrap(err, "compute relative path of current dir against $GOPATH/src")
	}

	if strings.HasPrefix(rel, "..") {
		return "", errors.Newf("current path %s is out of the GOPATH/src", curdir)
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	switch len(parts) {
	case 1, 2:
		if parts[0] == "" {
			return "", errors.New("you are right in the $GOPATH/src, no package here is allowed")
		}

		return parts[0], nil
	default:
		// take first three components as a package path in case if it is valid package domain name, otherwise take
		// the first component
		if checkPackageNameDomain(parts[0]) != nil {
			return parts[0], nil
		}

		return path.Join(parts[:3]...), nil
	}
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
