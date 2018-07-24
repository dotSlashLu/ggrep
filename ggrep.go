// go grep
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
)

type excludeCfg []string

func (i *excludeCfg) String() string {
	return strings.Join([]string(*i), ", ")
}

func (i *excludeCfg) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type cfg struct {
	// limit of searched bytes
	szlimit   int
	bufSize   int
	recursive bool
	// exclude glob
	exclude  excludeCfg
	parallel int
	dst      string
	pattern  string
}

const (
	gSzlimit   = 20
	gBufSize   = 10240
	gRecursive = false
)

var gCfg cfg

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
	gCfg.dst = "."
	if flag.NArg() > 1 {
		gCfg.dst = args[1]
	}
}

func readDir(dir string, tasks chan string) {
	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, f := range finfos {
		fname := path.Join(dir, f.Name())
		if f.IsDir() {
			if !gCfg.recursive {
				continue
			}
			// fmt.Println("is dir, Name():", fname)
			readDir(fname, tasks)
		} else {
			// fmt.Println("gened task", fname)
			tasks <- fname
		}
	}
}

func matchFile(fpath string) error {
	buf := make([]byte, gCfg.bufSize)
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	var boundary []byte
	for {
		_, err = f.Read(buf)
		// n, err := f.Read(buf)
		// fmt.Println("read ", n)
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			return nil
		}
		// TODO
		// this copies buf which is a huge perf drawback
		src := append(boundary, buf...)

		// find all indicies in buf that matches the pattern
		idx, fromI := 0, 0
		for {
			idx = strings.Index(string(src[fromI:]), gCfg.pattern)
			if idx == -1 {
				break
			}
			// TODO
			// should print file idx not idx from current buf
			fmt.Println(fpath, "matched", fromI+idx)
			fromI += idx + 1
		}
		// - 3 to ensure the utf-8 boundary is inside our boundary
		// - (len(pattern) - 1) ensures pattern not on boundary
		// fmt.Println("len src", len(src), "len patter", len(gCfg.pattern))
		boundaryI := len(src) - 3 - (len(gCfg.pattern) - 1)
		boundary = src[boundaryI:]
	}
	return nil
}

func main() {
	fmt.Println("gogrep")
	parseFlags()
	fmt.Printf("%+v\n", gCfg)

	tasks := make(chan string)
	// a worker pool
	var wg sync.WaitGroup
	for i := 0; i < gCfg.parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range tasks {
				matchFile(f)
			}
		}()
	}

	// if dst is a file, just match it
	dirinfo, err := os.Lstat(gCfg.dst)
	if err != nil {
		panic(err)
	}
	if !dirinfo.IsDir() {
		tasks <- gCfg.dst
	} else {
		readDir(gCfg.dst, tasks)
	}

	close(tasks)
	wg.Wait()
}
