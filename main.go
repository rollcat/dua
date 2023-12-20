package main

import (
	"fmt"
	_ "io/fs"
	"os"
	"path"
	"slices"
	"strconv"

	"github.com/rollcat/getopt"
)

var threshold float64 = 0.9
var topn int = 20

const (
	KB = 1024 << (iota * 10)
	MB
	GB
	TB
	PB
)

func showUsage() {
	println("Usage: dua [-h] [-t THRESHOLD] [-n N] <DIRECTORY>")
}

func showHelp() {
	showUsage()
	println(`
"dua" stands for "disk usage analyzer"; it scans the target directory
for files and directories taking up the most space.

Options:
    -h            Show this help and exit.
    -t THRESHOLD  Set the threshold (default: 0.9; range (0.0 - 1.0)).
    -n N          Show top N results (default: 20).
`)
}

func Eprintln(s string) (int, error) {
	return fmt.Fprintln(os.Stderr, s)
}

func fmtBytes[I ~int64 | uint64](i I) string {
	switch {
	case i < KB:
		return fmt.Sprintf("%7d  b", i)
	case i < MB:
		return fmt.Sprintf("%7.2f KB", float64(i)/KB)
	case i < GB:
		return fmt.Sprintf("%7.2f MB", float64(i)/MB)
	case i < TB:
		return fmt.Sprintf("%7.2f GB", float64(i)/GB)
	case i < PB:
		return fmt.Sprintf("%7.2f TB", float64(i)/TB)
	default:
		return fmt.Sprintf("%7.2f PB", float64(i)/PB)
	}
}

type NodeStat struct {
	path     string
	type_    string
	subtotal int64
	total    int64
	children []*NodeStat
}

func NewNodeStat(p string) *NodeStat {
	return &NodeStat{
		path:     p,
		type_:    " ",
		children: []*NodeStat{},
	}
}

func (s *NodeStat) String() string {
	return fmt.Sprintf("%s [%s] %s", fmtBytes(s.Total()), s.type_, s.path)
}

func (s *NodeStat) Walk() error {
	f, err := os.Open(s.path)
	if err != nil {
		Eprintln(err.Error())
		return err
	}
	dirEntries, err := f.ReadDir(-1)
	if err != nil {
		f.Close()
		Eprintln(err.Error())
		return err
	}
	f.Close()

	for _, d := range dirEntries {
		fpath := path.Join(s.path, d.Name())
		child := NewNodeStat(fpath)
		s.children = append(s.children, child)
		if d.IsDir() {
			child.type_ = "d"
			if err := child.Walk(); err != nil {
				continue
			}
		} else if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				Eprintln(err.Error())
				return err
			}
			child.type_ = "f"
			child.total = info.Size()
		} else {
			child.type_ = "?"
		}
	}
	return nil
}

func (s *NodeStat) Total() int64 {
	if s.total == 0 {
		s.total = s.subtotal
		for _, child := range s.children {
			s.total += child.Total()
		}
	}
	return s.total
}

func (s *NodeStat) Top(n uint) []*NodeStat {
	s.Total()
	top := []*NodeStat{}
	includeSelf := true
	for _, child := range s.children {
		// if any single child takes up more than a % of the total,
		// don't include self in the top stats (as self would compete
		// with the child for the top spot)
		if float64(child.Total()) > (float64(s.Total()) * threshold) {
			includeSelf = false
		}
		// include up to n top candidates from each child, since we're
		// not meant to return more than n anyway
		top = append(top, child.Top(n)...)
	}
	if includeSelf {
		top = append(top, s)
	}
	slices.SortFunc(top, func(a, b *NodeStat) int {
		return int(b.total - a.total)
	})
	if n > 0 {
		return top[:min(n, uint(len(top)))]
	} else {
		return top
	}
}

func main() {
	args, opts, err := getopt.GetOpt(
		os.Args[1:],
		"ht:n:",
		nil,
	)
	if err != nil {
		showUsage()
		os.Exit(1)
	}
	for _, opt := range opts {
		switch opt.Option {
		// case "-v":
		// 	showVersion()
		// 	os.Exit(0)
		case "-h":
			showHelp()
			os.Exit(0)
		case "-t":
			var err error
			if threshold, err = strconv.ParseFloat(opt.Argument, 64); err != nil {
				Eprintln(err.Error())
				os.Exit(1)
			}
			if !(0.0 < threshold && threshold < 1.0) {
				Eprintln("Threshold not in range (0.0 - 1.0)")
				os.Exit(1)
			}
		case "-n":
			var err error
			if topn, err = strconv.Atoi(opt.Argument); err != nil {
				Eprintln(err.Error())
				os.Exit(1)
			}
			if topn <= 0 {
				Eprintln("N must be greater than 0.")
				os.Exit(1)
			}
		default:
			panic("unexpected argument")
		}
	}
	if len(args) != 1 {
		showUsage()
		os.Exit(1)
	}

	root := NewNodeStat(args[0])
	if err := root.Walk(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
	// println(fmtBytes(root.Total()))
	for _, s := range root.Top(uint(topn)) {
		println(s.String())
	}
}
