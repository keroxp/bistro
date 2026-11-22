package app

import "github.com/urfave/cli/v3"

func rootFsFlag() (*string, *cli.StringFlag) {
	var rootFs string
	rootFsFlag := &cli.StringFlag{
		Name:        "rootfs",
		Usage:       "Root filesystem: tar:<path> or oci:<tar-path> from docker save (default: host filesystem)",
		Destination: &rootFs,
	}
	return &rootFs, rootFsFlag
}
