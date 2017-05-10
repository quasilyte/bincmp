package cmp

import (
	"regexp"

	"github.com/tzneal/bincmp/nm"
	"github.com/tzneal/bincmp/readelf"
)

// Comparer is used to determine the diferences between two binaries
type Comparer struct {
	fileA string
	fileB string
	o     Options
	w     Writer
}

type Options struct {
	Pattern string
	Writer  Writer
}

// NewComparer creates a comparer used to compare between binaries
// fileA and fileB
func NewComparer(fileA, fileB string, o Options) *Comparer {
	return &Comparer{
		fileA: fileA,
		fileB: fileB,
		o:     o,
		w:     o.Writer}
}

func (c *Comparer) CompareSymbols() error {
	aSyms, err := nm.ListSymbols(c.fileA)
	if err != nil {
		return err
	}
	bSyms, err := nm.ListSymbols(c.fileB)
	if err != nil {
		return err
	}

	aKnown, bKnown, symNames := uniqSymNames(aSyms, bSyms)

	first := true
	re := regexp.MustCompile(c.o.Pattern)
	for _, name := range symNames {
		if !re.MatchString(name) {
			continue
		}
		if aKnown[name].Size == bKnown[name].Size {
			continue
		}

		if first {
			c.w.StartSymbols()
			defer c.w.EndSymbols()
			first = false
		}
		if err := c.w.WriteSymbol(aKnown[name], bKnown[name]); err != nil {
			return err
		}
	}

	return nil
}

func (c *Comparer) CompareSections() error {
	aSects, err := readelf.ListSections(c.fileA)
	if err != nil {
		return err
	}
	bSects, err := readelf.ListSections(c.fileB)
	if err != nil {
		return err
	}

	aKnown, bKnown, sectNames := uniqSectNames(aSects, bSects)

	re := regexp.MustCompile(c.o.Pattern)
	first := true
	for _, name := range sectNames {
		if !re.MatchString(name) {
			continue
		}

		if aKnown[name].Size == bKnown[name].Size {
			continue
		}
		if first {
			first = false
			c.w.StartSections()
			defer c.w.EndSections()
		}
		if err := c.w.WriteSection(aKnown[name], bKnown[name]); err != nil {
			return err
		}
	}

	return nil
}