package fancyfmt

// ImportsGrouper a facility to give an import path a certain weight what will be used to group a set of imports by it,
// i.e. import paths with the same weight will be in one groups of imports
type ImportsGrouper interface {
	Weight(path string) int
}
