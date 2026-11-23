package app

import (
	"context"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/urfave/cli/v3"
)

func (c *app) bundleCommand() *cli.Command {
	rootFsPtr, rootFsFlag := rootFsFlag()
	var dest bistro.BundleInput
	return &cli.Command{
		Name:      "bundle",
		ArgsUsage: "BUNDLE [files or directories...]",
		ArgValidator: func(ctx context.Context, cmd *cli.Command) error {
			return c.validatePathArray(&dest.Files)(cmd.Args().Slice())
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "out",
				Aliases:  []string{"o"},
				Usage:    "Output root filesystem tar archive",
				Required: true,
				Validator: func(value string) error {
					v, err := rootfs.NewPath(c.absPath(value))
					if err != nil {
						return err
					}
					dest.Out = v
					return nil
				},
			},
			&cli.StringSliceFlag{
				Name:        "package",
				Usage:       "Packages to include in the root filesystem",
				Destination: &dest.Packages,
			},
			&cli.StringSliceFlag{
				Name:      "exclude",
				Usage:     "Files or directories to exclude from the root filesystem.",
				Validator: c.validatePathArray(&dest.Excludes),
			},
			&cli.StringSliceFlag{
				Name:      "exclude-scan",
				Usage:     "Files or directories to exclude from scanning.",
				Validator: c.validatePathArray(&dest.ExcludeScans),
			},
			rootFsFlag,
			&cli.BoolFlag{
				Name:        "no-scan",
				Usage:       "Disable scanning for ELF dependencies",
				Destination: &dest.NoScan,
			},
			&cli.StringFlag{
				Name:      "groups",
				Usage:     "Groups to include in the root filesystem (JSON format)",
				Validator: validateJsonArray(&dest.Groups),
			},
			&cli.StringFlag{
				Name:      "users",
				Usage:     "Users to include in the root filesystem (JSON format)",
				Validator: validateJsonArray(&dest.Users),
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
			return bistro.Bundle(di, dest)
		},
	}
}
