package bistro

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/loilo-inc/bistro/pkg/docker"
	"github.com/stretchr/testify/require"
)

type importClient struct {
	docker.Client
	importFunc func(context.Context, io.Reader, docker.ImportOptions) (string, error)
}

func (c importClient) Import(ctx context.Context, src io.Reader, opts docker.ImportOptions) (string, error) {
	return c.importFunc(ctx, src, opts)
}

func TestMakerImportsCompletedBundle(t *testing.T) {
	dir := rootFSTempDir(t)
	file := filepath.Join(dir, "payload.txt")
	writeRootFSFile(t, file, "image payload", 0644)
	var archivePath string
	client := importClient{importFunc: func(ctx context.Context, src io.Reader, opts docker.ImportOptions) (string, error) {
		archivePath = src.(*os.File).Name()
		data, err := io.ReadAll(src)
		require.NoError(t, err)
		entries := readRootFSTar(t, data)
		require.Equal(t, "image payload", entries[rootFSArchiveName(file)].body)
		require.Equal(t, "root:x:0:0:root:/root:/sbin/nologin\n", entries["etc/passwd"].body)
		require.Equal(t, docker.ImportOptions{
			Repository: "example/test", Tag: "dev", Platform: "linux/arm64",
			Changes: []string{"WORKDIR /app"},
		}, opts)
		return "sha256:example", nil
	}}
	maker := NewMaker(DI{FS: os.DirFS("/")}, client)
	id, err := maker.Make(&Recipe{
		Build:  BuildConfig{Workdir: "/app"},
		Bundle: BundleConfig{Files: []string{file}},
	}, MakeInput{Repository: "example/test", Tag: "dev", Platform: "linux/arm64"})
	require.NoError(t, err)
	require.Equal(t, "sha256:example", id)
	require.NotEmpty(t, archivePath)
	_, err = os.Stat(archivePath)
	require.True(t, os.IsNotExist(err), "temporary archive was not removed: %v", err)
}
