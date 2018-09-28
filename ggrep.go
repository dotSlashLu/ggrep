// go grep
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
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
	exclude    excludeCfg
	parallel   int
	dsts       []string
	rePattern  *regexp.Regexp
	pattern    string
	debug      bool
	stringMode bool
}

const (
	gSzlimit   = 20
	gBufSize   = 10240
	gRecursive = false
)

var gCfg cfg

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
			readDir(fname, tasks)
		} else {
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
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			return nil
		}

		// TODO this might be very wrong
		//	encodings like JIS and GBK might not be valid UTF8 and apparently
		// 	they are not binary
		isBinary := !utf8.Valid(buf)
		if isBinary {
			if gCfg.debug {
				fmt.Println(fpath, "is not utf8(compatible), abandon")
			}
			return nil
		}

		// TODO this copies buf which is a huge perf drawback
		src := append(boundary, buf...)

		if gCfg.stringMode {
			// exact string match
			// find all indicies in buf that matches the pattern
			idx, fromI := 0, 0
			for {
				idx = strings.Index(string(src[fromI:]), gCfg.pattern)
				if idx == -1 {
					break
				}
				// TODO should print file idx not idx from current buf
				fmt.Println(fpath, "matched", fromI+idx)
				fromI += idx + 1
			}
		} else {
			// regex match
			matched := gCfg.rePattern.FindAllIndex(src, -1)
			if matched != nil {
				fmt.Println(fpath, "matched indicies", matched)
			}
		}

		// - 4 to ensure the utf-8 boundary is inside our boundary
		// - (len(pattern) - 1) ensures pattern not on boundary
		// fmt.Println("len src", len(src), "len patter", len(gCfg.pattern))
		boundaryI := len(src) - 4 - (len(gCfg.pattern) - 1)
		boundary = src[boundaryI:]
	}
	return nil
}

func main() {
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
				// fmt.Println("got task", f)
				matchFile(f)
			}
		}()
	}

	for _, dst := range gCfg.dsts {
		// if dst is a file, just match it
		dirinfo, err := os.Lstat(dst)
		if err != nil {
			panic(err)
		}
		if !dirinfo.IsDir() {
			tasks <- dst
		} else {
			readDir(dst, tasks)
		}
	}

	close(tasks)
	wg.Wait()
}
