package rootfs

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/loilo-inc/bistro/pkg/util"
)

func NewPathSet(list []Path) PathSet {
	result := PathSet{origPaths: util.NewOrderedSet(list...)}
	return result
}

type PathSet struct {
	origPaths     util.OrderedSet[Path]
	resolvedPaths util.Map[Path, *Path]
}

type PathMatch struct {
	rule    Path
	real    *Path
	matched Path
}

func (e *PathMatch) Rule() Path {
	return e.rule
}

func (e *PathMatch) Matched() Path {
	return e.matched
}

func (e *PathMatch) Real() *Path {
	return e.real
}

type ForbiddenPathError struct {
	PathMatch
}

func (e *ForbiddenPathError) Error() string {
	if e.real != nil {
		return "forbidden: " + e.rule.String() + " (matched: " + e.matched.String() + ", real: " + e.real.String() + ")"
	}
	return "forbidden: " + e.rule.String() + " (matched: " + e.matched.String() + ")"
}

func pathMatch(p Path, tgt Path) bool {
	if p.Abs() == tgt.Abs() ||
		strings.HasPrefix(p.Abs(), strings.TrimSuffix(tgt.Abs(), "/")+"/") {
		return true
	}
	return false
}

func (ps *PathSet) matchOrig(p Path) *PathMatch {
	for rule := range ps.origPaths.Values() {
		if pathMatch(p, rule) {
			return &PathMatch{rule: rule, matched: p}
		}
	}
	return nil
}

func (ps *PathSet) matchResolved(p Path) *PathMatch {
	for orig := range ps.origPaths.Values() {
		res := ps.resolvedPaths.Get(orig)
		if res == nil {
			continue
		}
		if m := pathMatch(p, *res); m {
			return &PathMatch{rule: orig, matched: p, real: res}
		}
	}
	return nil
}

// Resolve resolves the original paths to their real paths using the provided filesystem.
func (ps *PathSet) Resolve(fss fs.FS) error {
	if ps.isResolved() {
		return nil
	}
	for p := range ps.origPaths.Values() {
		if ps.resolvedPaths.Has(p) {
			continue
		}
		dir := p.Dir()
		realDir, err := dir.RealPath(fss)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			ps.resolvedPaths.Set(p, nil)
		case err != nil:
			return err
		case realDir.Abs() == dir.Abs():
			ps.resolvedPaths.Set(p, nil)
		default:
			res := realDir.Join(p.Base())
			ps.resolvedPaths.Set(p, &res)
		}
	}
	return nil
}

// Match checks if the given path matches any of the original or resolved paths in the set.
func (ps *PathSet) Match(fss fs.FS, p Path) (*PathMatch, error) {
	if ps.origPaths.Size() == 0 {
		return nil, nil
	}
	if m := ps.matchOrig(p); m != nil {
		return m, nil
	}
	if !ps.isResolved() {
		if err := ps.Resolve(fss); err != nil {
			return nil, err
		}
	}
	if m := ps.matchResolved(p); m != nil {
		return m, nil
	}

	var matched *PathMatch
	onStart := func(p Path) (bool, error) {
		if m := ps.matchOrig(p); m != nil {
			matched = m
			return false, nil
		}
		if m := ps.matchResolved(p); m != nil {
			matched = m
			return false, nil
		}
		return true, nil
	}
	dest, err := newSymlinkVisitor(fss, onStart, nil).Resolve(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	case dest == nil:
		return matched, nil
	default:
		if m := ps.matchOrig(*dest); m != nil {
			return m, nil
		}
		if m := ps.matchResolved(*dest); m != nil {
			return m, nil
		}
		return nil, nil
	}
}

func (ps *PathSet) isResolved() bool {
	return ps.origPaths.Size() == ps.resolvedPaths.Size()
}

func (ps *PathSet) Clone() PathSet {
	return PathSet{
		origPaths:     ps.origPaths.Clone(),
		resolvedPaths: ps.resolvedPaths.Clone(),
	}
}

func (ps *PathSet) Add(p ...Path) {
	ps.origPaths.Add(p...)
}

func (ps *PathSet) Remove(p Path) {
	ps.origPaths.Remove(p)
	ps.resolvedPaths.Delete(p)
}
