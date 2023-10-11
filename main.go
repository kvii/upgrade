// Upgrade provide a tool to upgrade apps installed by "go install".
//
// Usage:
//
//	# upgrade app in ~/go/bin
//	upgrade ~/go/bin
//
//	# upgrade multiple apps
//	upgrade ~/go/bin/foo ~/go/bin/bar
//
//	# apply flags to "go install"
//	upgrade ~/go/bin/foo -- -v
package main

import (
	"context"
	"debug/buildinfo"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
)

// flags
var (
	x bool
)

func init() {
	flag.BoolVar(&x, "x", false, "print commands that executed")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	args, goArgs := cut(flag.Args(), "--")
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "no arguments")
		os.Exit(1)
	}
	for _, p := range args {
		err := upgrade(ctx, p, goArgs)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func upgrade(ctx context.Context, name string, goArgs []string) error {
	stat, err := os.Stat(name)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return upgradeDir(ctx, name, goArgs)
	}
	return upgradeFile(ctx, name, goArgs)
}

func upgradeDir(ctx context.Context, name string, goArgs []string) error {
	return filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == name {
			return nil
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		return upgradeFile(ctx, path, goArgs)
	})
}

func upgradeFile(ctx context.Context, name string, goArgs []string) error {
	bi, err := buildinfo.ReadFile(name)
	if err != nil {
		return err
	}

	// args = {"install", ...goArgs, "xx@latest"}
	args := make([]string, len(goArgs)+2)
	args[0] = "install"
	copy(args[1:], goArgs)
	args[len(args)-1] = bi.Path + "@latest"

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if x {
		fmt.Println(cmd)
	}
	return cmd.Run()
}

func cut(a []string, s string) (before, after []string) {
	before = make([]string, 0, len(a))
	after = make([]string, 0, len(a))

	var b bool
	for _, v := range a {
		switch {
		case v == s && !b:
			b = true
		case b:
			after = append(after, v)
		default:
			before = append(before, v)
		}
	}
	return
}
