package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

var sep = flag.String("sep", "_", "separator character(s) for last part of file names")

func main() {
	var prefix string
	var indx int

	flag.Parse()
	files := flag.Args()
	sort.Strings(files)

	for _, orig := range files {
		i := strings.LastIndexByte(orig, (*sep)[0])
		if i < 0 {
			fmt.Println(orig, ": ignoring, no separator found")
			continue
		}
		if prefix != orig[:i] {
			prefix = orig[:i]
			indx = 0
		}
		dot := strings.IndexByte(orig[i+1:], '.')
		if dot < 0 {
			fmt.Println(orig, ": missing dot after last ", sep)
			continue
		}
		dot += i + 1
		repl := fmt.Sprintf("%s_%04d%s", prefix, indx, orig[dot:])
		if err := os.Rename(orig, repl); err != nil {
			fmt.Println(err)
		} else {
			indx++
		}
	}
}
