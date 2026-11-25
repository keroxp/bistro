package app

import (
	"encoding/json"

	"github.com/loilo-inc/bistro/pkg/bistro"
	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/urfave/cli/v3"
)

func rootFsFlag() (*string, *cli.StringFlag) {
	var rootFs string
	rootFsFlag := &cli.StringFlag{
		Name:        "rootfs",
		Usage:       "Root filesystem: tar:<path> or oci:<tar-path> from docker save (default: host filesystem)",
		Destination: &rootFs,
	}
	return &rootFs, rootFsFlag
}

func decodeJsonArray[T any](arr string) ([]T, error) {
	var result []T
	if err := json.Unmarshal([]byte(arr), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func validateJsonArray[T bistro.Validator](dest *[]T) func(string) error {
	return func(value string) error {
		v, err := decodeJsonArray[T](value)
		if err != nil {
			return err
		}
		for _, item := range v {
			if err := item.Validate(); err != nil {
				return err
			}
		}
		*dest = v
		return nil
	}
}

func (c *app) validatePathArray(dest *[]rootfs.Path) func([]string) error {
	return func(values []string) error {
		paths := make([]rootfs.Path, len(values))
		for i, v := range values {
			v, err := rootfs.NewPath(c.absPath(v))
			if err != nil {
				return err
			}
			paths[i] = v
		}
		*dest = paths
		return nil
	}
}
