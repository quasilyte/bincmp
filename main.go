package main

import (
	"bufio"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type symbols map[string]int64

func main() {
	disFlag := flag.Bool("disassemble", false, "if true, display disassembly of non-matching functions")
	flag.Parse()
	f1 := flag.Arg(0)
	f2 := flag.Arg(1)

	f1Sym := parseSyms(f1)
	f2Sym := parseSyms(f2)
	var f1Dis, f2Dis dsyms
	if *disFlag {
		f1Dis = disassemble(f1)
		f2Dis = disassemble(f2)
	}

	delta := int64(0)
	fmt.Println("# delta name sz1 sz2")
	for name, sz := range f1Sym {
		if sz2, ok := f2Sym[name]; ok {
			// removing from maps so we can determine and print out the
			// symbols found in only one of the binaries
			delete(f1Sym, name)
			delete(f2Sym, name)
			if sz == sz2 {
				continue
			}
			fmt.Printf("%d %s %d %d\n", sz-sz2, name, sz, sz2)
			delta += sz - sz2
			if *disFlag {
				dump(name, f1Dis, f2Dis)
			}
		}
	}

	// any remaining symbols must only be in one of the files, so identify them
	for name := range f1Sym {
		fmt.Printf("-%s\n", name)
	}
	for name := range f2Sym {
		fmt.Printf("+%s\n", name)
	}

	// finally print out a size summary
	if delta > 0 {
		fmt.Printf("%s is bigger than %s [%d]\n", f1, f2, delta)
	} else if delta < 0 {
		fmt.Printf("%s is smaller than %s [%d]\n", f1, f2, delta)
	}

}

// run executes a process and returns a scanner that allows parsing stdout line
// by line.
func run(args ...string) *bufio.Scanner {
	cmd := exec.Command(args[0], args[1:]...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	return bufio.NewScanner(cmdReader)
}

// parseSyms runs nm to identify the size of symbols.
func parseSyms(fn string) symbols {
	syms := symbols{}
	scanner := run("nm", "-S", "--size-sort", fn)
	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		// address size type name
		if len(line) != 4 {
			continue
		}
		sz, err := strconv.ParseInt(line[1], 16, 64)
		if err == nil {
			syms[line[3]] = sz
		}
	}
	return syms
}

type dsym struct {
	code   []string
	maxLen int
}
type dsyms map[string]*dsym

// disassemble runs objdump to disassemble the binary and creates a map
// of symbol to disassembled code.
func disassemble(fn string) dsyms {
	scanner := run("objdump", "-d", "--no-show-raw-insn", fn)
	ds := make(dsyms)
	// regexp for maching the start of disassembly for a symbol
	startDis, err := regexp.Compile("^[0-9a-f]+ <(.*?)>:$")
	if err != nil {
		panic(err)
	}

	var lastSym string
	for scanner.Scan() {
		match := startDis.FindStringSubmatch(scanner.Text())
		if len(match) > 0 {
			lastSym = match[1]
			continue
		}
		if len(lastSym) > 0 {
			if _, ok := ds[lastSym]; !ok {
				ds[lastSym] = &dsym{}
			}
			sym := ds[lastSym]
			// TODO: Parse the output of objdump
			code := strings.Replace(scanner.Text(), "\t", "    ", -1)
			sym.code = append(sym.code, code)
			if len(code) > sym.maxLen {
				sym.maxLen = len(code)
			}
		}
	}
	return ds
}

// dump prints out a side by side listing of the disassembly captured
// from objdump from each binary.
func dump(sym string, s1, s2 dsyms) {
	f1 := s1[sym]
	f2 := s2[sym]
	if f1 == nil || f2 == nil {
		return
	}
	for i, j := 0, 0; i < len(f1.code) && j < len(f2.code); {
		if i < len(f1.code) {
			fmt.Printf("%s", f1.code[i])
		}
		// pad to the same length so the rhs listing will be aligned
		printSpaces(f1.maxLen - len(f1.code[i]))
		if j < len(f2.code) {
			fmt.Printf("%s", f2.code[j])
		}
		fmt.Printf("\n")

		i++
		j++
	}
}
func printSpaces(n int) {
	for i := 0; i < n; i++ {
		fmt.Print(" ")
	}
}