package app

import (
	"context"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/urfave/cli/v3"
)

func (c *app) bundleCommand() *cli.Command {
	rootFsPtr, rootFsFlag := rootFsFlag()
	var packages []string
	var excludes []string
	var excludeScans []string
	var noScan bool
	return &cli.Command{
		Name:      "bundle",
		ArgsUsage: "BUNDLE [files or directories...]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "out",
				Aliases:  []string{"o"},
				Usage:    "Output root filesystem tar archive",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:        "package",
				Usage:       "Packages to include in the root filesystem",
				Destination: &packages,
			},
			&cli.StringSliceFlag{
				Name:        "exclude",
				Usage:       "Files or directories to exclude from the root filesystem.",
				Destination: &excludes,
			},
			&cli.StringSliceFlag{
				Name:        "exclude-scan",
				Usage:       "Files or directories to exclude from scanning.",
				Destination: &excludeScans,
			},
			rootFsFlag,
			&cli.BoolFlag{
				Name:        "no-scan",
				Usage:       "Disable scanning for ELF dependencies",
				Destination: &noScan,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fsHandle, err := openFs(*rootFsPtr)
			if err != nil {
				return err
			}
			defer fsHandle.Close()
			fss := fsHandle.FS()
			di := c.depsFromFs(fss)
			if err := load(di); err != nil {
				return err
			}
			return c.bundle(di, bistro.BundleInput{
				Out:          rootfs.MustPath(c.absPath(cmd.String("out"))),
				Files:        c.rootfsPaths(cmd.Args().Slice()),
				Packages:     packages,
				Excludes:     c.rootfsPaths(excludes),
				ExcludeScans: c.rootfsPaths(excludeScans),
				NoScan:       noScan,
			})
		},
	}
}

func (c *app) bundle(di bistro.DI, input bistro.BundleInput) error {
	return bistro.Bundle(di, input)
}
