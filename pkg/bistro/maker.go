package bistro

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/loilo-inc/bistro/pkg/docker"
	"github.com/loilo-inc/bistro/pkg/rootfs"
)

type Maker interface {
	Make(recipe Recipe, repository string, tag string) (string, error)
}

type maker struct {
	DI
	docker docker.Client
}

func NewMaker(di DI, docker docker.Client) *maker {
	return &maker{DI: di, docker: docker}
}

func toRootFsPaths(paths []string) ([]rootfs.Path, error) {
	result := make([]rootfs.Path, 0, len(paths))
	for _, p := range paths {
		result = append(result, rootfs.MustPath(p))
	}
	return result, nil
}

type MakeInput struct {
	Repository string
	Tag        string
	Platform   string
}

func (m *maker) Make(recipe *Recipe, input MakeInput) (string, error) {
	bundle := recipe.Bundle
	outDir, err := os.MkdirTemp("", "bundle")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(outDir)
	outPath := filepath.Join(outDir, "rootfs.tar")
	bundleInput := BundleInput{
		Out:      rootfs.MustPath(outPath),
		Packages: bundle.Packages,
		Users:    bundle.Users,
		Groups:   bundle.Groups,
	}
	if files, err := toRootFsPaths(bundle.Files); err != nil {
		return "", err
	} else {
		bundleInput.Files = files
	}
	if excludes, err := toRootFsPaths(bundle.Excludes); err != nil {
		return "", err
	} else {
		bundleInput.Excludes = excludes
	}
	if excludeScans, err := toRootFsPaths(bundle.ExcludeScans); err != nil {
		return "", err
	} else {
		bundleInput.ExcludeScans = excludeScans
	}
	if err := Bundle(m.DI, bundleInput); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Bundle atomically replaces its output; open the completed archive afterward.
	outFile, err := os.Open(outPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	changes, err := Encode(&recipe.Build)
	if err != nil {
		return "", err
	}
	imageId, err := m.docker.Import(ctx, outFile, docker.ImportOptions{
		Repository: input.Repository,
		Tag:        input.Tag,
		Platform:   input.Platform,
		Changes:    changes,
	})
	if err != nil {
		return "", err
	}
	return imageId, nil
}

func Encode(b *BuildConfig) ([]string, error) {
	var result []string
	if b.User != "" {
		result = append(result, fmt.Sprintf("USER %s", b.User))
	}
	if b.StopSignal != "" {
		result = append(result, fmt.Sprintf("STOPSIGNAL %s", b.StopSignal))
	}
	if b.Workdir != "" {
		result = append(result, fmt.Sprintf("WORKDIR %s", b.Workdir))
	}
	if len(b.Volumes) > 0 {
		b, err := json.Marshal(b.Volumes)
		if err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf("VOLUME %s", b))
	}
	for _, label := range b.Labels {
		result = append(result, fmt.Sprintf("LABEL %s=%s", label.Name, label.Value))
	}
	for _, env := range b.Envs {
		result = append(result, fmt.Sprintf("ENV %s=%s", env.Name, env.Value))
	}
	if len(b.Cmd) > 0 {
		b, err := json.Marshal(b.Cmd)
		if err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf("CMD %s", b))
	}
	if len(b.Entrypoint) > 0 {
		b, err := json.Marshal(b.Entrypoint)
		if err != nil {
			return nil, err
		}
		result = append(result, fmt.Sprintf("ENTRYPOINT %s", b))
	}
	return result, nil
}
