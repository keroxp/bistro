package app

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/dpkg"
	"github.com/loilo-inc/bistro/pkg/ldd"
	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/loilo-inc/bistro/pkg/util"
	"github.com/urfave/cli/v3"
)

type scanApp struct {
	*app
	bistro.DI
	scanAppInput
}

type scanAppInput struct {
	paths    []rootfs.Path
	excludes []rootfs.Path
	format   string
	fs       fs.FS
}

func (c *app) scanCommand() *cli.Command {
	rootFsPtr, rootFsFlag := rootFsFlag()
	var files []rootfs.Path
	var excludes []rootfs.Path
	var format string
	return &cli.Command{
		Name:      "scan",
		ArgsUsage: "SCAN [fileorDirs...]",
		ArgValidator: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("scan requires at least one file or directory")
			}
			return c.validatePathArray(&files)(cmd.Args().Slice())
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "format",
				Aliases:     []string{"f"},
				Usage:       "Output format (list, flat, tree)",
				Value:       "tree",
				Destination: &format,
				Validator: func(value string) error {
					if value != "list" && value != "flat" && value != "tree" {
						return fmt.Errorf("invalid format: %s", value)
					}
					return nil
				},
			},
			rootFsFlag,
			&cli.StringSliceFlag{
				Name:      "exclude",
				Usage:     "Files or directories to exclude from scanning.",
				Validator: c.validatePathArray(&excludes),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fsOpener, err := openFs(*rootFsPtr)
			if err != nil {
				return err
			}
			defer fsOpener.Close()
			fss := fsOpener.FS()
			return c.newScanApp(scanAppInput{
				paths:    files,
				excludes: excludes,
				format:   format,
				fs:       fss,
			}).Run(ctx)
		},
	}
}

func (c *app) newScanApp(input scanAppInput) *scanApp {
	return &scanApp{
		app:          c,
		DI:           c.depsFromFs(input.fs),
		scanAppInput: input,
	}
}

func (c *scanApp) Run(ctx context.Context) error {
	if err := load(c.DI); err != nil {
		return err
	}
	switch c.format {
	case "list":
		return c.scanList(c.paths, c.excludes, false)
	case "flat":
		return c.scanList(c.paths, c.excludes, true)
	case "tree":
		return c.scanTree(c.paths, c.excludes)
	}
	return fmt.Errorf("unknown format: %s", c.format)
}

func (c *scanApp) scan(paths []rootfs.Path, excludes []rootfs.Path) ([]*bistro.ElfDependency, error) {
	scanner := bistro.NewScanner(c.DI)
	scanner.SetExcludes(rootfs.NewPathSet(excludes))
	var elfDeps []*bistro.ElfDependency
	for _, path := range paths {
		deps, err := scanner.Scan(path)
		if err != nil {
			return nil, err
		}
		elfDeps = append(elfDeps, deps...)
	}
	return elfDeps, nil
}

func (c *scanApp) scanList(paths []rootfs.Path, excludes []rootfs.Path, flat bool) error {
	list, err := c.scan(paths, excludes)
	if err != nil {
		return err
	}
	pkgs := util.Map[string, *dpkg.Package]{}
	libs := util.Map[string, *ldd.Library]{}
	joinDeps := func() string {
		var deps []string
		for _, pkg := range pkgs.Iter() {
			deps = append(deps, pkg.NameWithArch())
		}
		for _, lib := range libs.Iter() {
			deps = append(deps, lib.Path.Abs())
		}
		return strings.Join(deps, " ")
	}
	for _, elfDep := range list {
		for _, exeDep := range elfDep.Dependencies {
			if exeDep.Package != nil {
				pkgs.Set(exeDep.Package.NameWithArch(), exeDep.Package)
			} else {
				libs.Set(exeDep.Library.Path.Abs(), exeDep.Library)
			}
		}
		for _, broken := range elfDep.BrokenLibs {
			c.errorf("[BROKEN] %s\n", broken)
		}
		var deps []string
		for _, pkg := range pkgs.Iter() {
			deps = append(deps, pkg.NameWithArch())
		}
		for _, lib := range libs.Iter() {
			deps = append(deps, lib.Path.Abs())
		}
		if !flat {
			c.printf("%s: %s\n", elfDep.ElfFile.Path, joinDeps())
			pkgs.Clear()
			libs.Clear()
		}
	}
	if flat {
		c.printf("%s\n", joinDeps())
	}
	return nil
}

func (c *scanApp) scanTree(paths []rootfs.Path, excludes []rootfs.Path) error {
	elfDeps, err := c.scan(paths, excludes)
	if err != nil {
		return err
	}
	for _, elfDep := range elfDeps {
		c.printf("%s\n", elfDep.ElfFile.Path)
		for _, exeDep := range elfDep.Dependencies {
			pkgName := "<none>"
			if exeDep.Package != nil {
				pkgName = exeDep.Package.NameWithArch()
			}
			c.printf("  %s => %s\n", exeDep.Library.Path, pkgName)
		}
		for _, broken := range elfDep.BrokenLibs {
			c.errorf("  [BROKEN] %s\n", broken)
		}
	}
	return nil
}
