package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
)

func parseFlags() {
	flag.IntVar(&gCfg.szlimit, "l", gSzlimit, "limit of file size")
	flag.IntVar(&gCfg.bufSize, "b", gBufSize, "buffer size in byte for each "+
		" file in parallel")
	flag.BoolVar(&gCfg.recursive, "r", gRecursive, "recursive")
	numCPU := runtime.NumCPU()
	flag.IntVar(&gCfg.parallel, "p", numCPU, "how many files to match in "+
		"parallel")
	flag.Var(&gCfg.exclude, "x",
		"exclude glob, can have multiple values like -x *.md -x .git")
	flag.Parse()

	// TODO: support multiple dst path
	if flag.NArg() == 0 {
		fmt.Printf("Usage: %s [options] pattern [path]\n", os.Args[0])
		fmt.Println("path: the file or path to search, default is the " +
			"current path\n\n")
		fmt.Println("options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	args := flag.Args()
	gCfg.pattern = args[0]
	gCfg.rePattern = regexp.MustCompile(args[0])
	gCfg.dst = "."
	if flag.NArg() > 1 {
		gCfg.dst = args[1]
	}
}
