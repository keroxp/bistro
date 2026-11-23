package rootfs

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestArchiverForbiddenPaths(t *testing.T) {
	for _, tc := range []struct{ name, rule, target string }{
		{"exact", "/real/cache/file", "/real/cache/file"},
		{"descendant", "/real/cache", "/real/cache/file"},
		{"canonical target", "/alias/cache", "/real/cache/file"},
		{"alias target", "/real/cache", "/other/cache/file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, operation := range []string{"copy", "recursive copy", "add"} {
				t.Run(operation, func(t *testing.T) {
					var out bytes.Buffer
					a := NewArchiver(pathSetFixture(), &out)
					a.Forbid(MustPath(tc.rule))
					var err error
					switch operation {
					case "copy":
						err = a.Copy(MustPath(tc.target))
					case "recursive copy":
						err = a.CopyRecursive(MustPath(tc.target))
					case "add":
						err = a.Add(MustPath(tc.target), []byte("generated"), 0644)
					}
					var forbidden *ForbiddenPathError
					require.ErrorAs(t, err, &forbidden)
					require.Equal(t, MustPath(tc.rule), forbidden.Rule())
					require.Contains(t, forbidden.Error(), tc.rule)
					require.NoError(t, a.Close())
				})
			}
		})
	}
}

func TestArchiverUnforbidAllowsGeneratedFile(t *testing.T) {
	fss := fstest.MapFS{"app": {Data: []byte("app")}}
	var out bytes.Buffer
	a := NewArchiver(fss, &out)
	passwd := MustPath("/etc/passwd")
	a.Forbid(passwd)
	require.NoError(t, a.Copy(MustPath("/app")))
	a.Unforbid(passwd)
	require.NoError(t, a.Add(passwd, []byte("generated"), 0644))
	require.NoError(t, a.Close())
	entries := readRootFSTar(t, out.Bytes())
	require.Equal(t, "app", entries["app"].body)
	require.Equal(t, "generated", entries["etc/passwd"].body)
}

func TestArchiverForbidTakesPriorityOverExclude(t *testing.T) {
	var out bytes.Buffer
	a := NewArchiver(pathSetFixture(), &out)
	file := MustPath("/real/keep")
	a.Exclude(file)
	a.Forbid(file)
	var forbidden *ForbiddenPathError
	require.ErrorAs(t, a.Copy(file), &forbidden)
	a.Unforbid(file)
	require.NoError(t, a.Copy(file))
	a.Unexclude(file)
	require.NoError(t, a.Copy(file))
	require.NoError(t, a.Close())
	require.Equal(t, "keep", readRootFSTar(t, out.Bytes())["real/keep"].body)
}

func TestArchiverUnexcludeRemovesResolvedRule(t *testing.T) {
	fss := fstest.MapFS{
		"alias":     {Mode: fs.ModeSymlink | 0777, Data: []byte("real")},
		"real/file": {Data: []byte("keep")},
	}
	var out bytes.Buffer
	a := NewArchiver(fss, &out)
	a.Exclude(MustPath("/alias/file"))
	require.NoError(t, a.Copy(MustPath("/real/file")))
	a.Unexclude(MustPath("/alias/file"))
	require.NoError(t, a.Copy(MustPath("/real/file")))
	require.NoError(t, a.Close())
	require.Equal(t, "keep", readRootFSTar(t, out.Bytes())["real/file"].body)
}
