package app

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/stretchr/testify/require"
)

func TestBundleUserGroupFlags(t *testing.T) {
	for _, tc := range []struct {
		name, users, groups, passwd, group string
	}{
		{
			name: "empty arrays", users: "[]", groups: "[]",
		},
		{
			name: "groups only", users: "[]",
			groups: `[{"name":"app","gid":1000},{"name":"worker","gid":1001}]`,
			group:  "app:x:1000:\nworker:x:1001:\n",
		},
		{
			name:   "multiple users and groups",
			users:  `[{"name":"app","uid":1000,"gid":1000},{"name":"worker","uid":1001,"gid":1001}]`,
			groups: `[{"name":"app","gid":1000},{"name":"worker","gid":1001}]`,
			passwd: "app:x:1000:1000::/home/app:/sbin/nologin\nworker:x:1001:1001::/home/worker:/sbin/nologin\n",
			group:  "app:x:1000:\nworker:x:1001:\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAppFixture(t)
			fixture.files["workspace/app"] = &fstest.MapFile{Data: []byte("app")}
			out := filepath.Join(t.TempDir(), "bundle.tar")
			require.NoError(t, fixture.run("bundle", "--out", out,
				"--users", tc.users, "--groups", tc.groups, "app"))
			want := map[string]string{"workspace/app": "app"}
			if tc.passwd != "" {
				want["etc/passwd"] = "root:x:0:0:root:/root:/sbin/nologin\n" + tc.passwd
			}
			if tc.group != "" {
				want["etc/group"] = "root:x:0:\n" + tc.group
			}
			require.Equal(t, want, readArchiveFiles(t, out))
		})
	}
}

func TestBundlePreservesExistingAccountsWithoutFlags(t *testing.T) {
	fixture := newAppFixture(t)
	fixture.files["etc/passwd"] = &fstest.MapFile{Data: []byte("existing:x:1000:1000::/:/bin/sh\n")}
	fixture.files["etc/group"] = &fstest.MapFile{Data: []byte("existing:x:1000:\n")}
	out := filepath.Join(t.TempDir(), "bundle.tar")
	require.NoError(t, fixture.run("bundle", "--out", out, "/etc"))
	require.Equal(t, map[string]string{
		"etc/passwd": "existing:x:1000:1000::/:/bin/sh\n",
		"etc/group":  "existing:x:1000:\n",
	}, readArchiveFiles(t, out))
}

func TestBundleRejectsExistingAccountConflicts(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		flags      []string
	}{
		{"passwd", "etc/passwd", []string{
			"--users", `[{"name":"app","uid":1000,"gid":1000}]`,
			"--groups", `[{"name":"app","gid":1000}]`,
		}},
		{"group", "etc/group", []string{"--groups", `[{"name":"app","gid":1000}]`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAppFixture(t)
			fixture.files[tc.file] = &fstest.MapFile{Data: []byte("existing\n")}
			dir := t.TempDir()
			out := filepath.Join(dir, "bundle.tar")
			require.NoError(t, os.WriteFile(out, []byte("original"), 0644))
			args := append([]string{"bundle", "--out", out}, tc.flags...)
			args = append(args, "/etc")
			err := fixture.run(args...)
			var forbidden *rootfs.ForbiddenPathError
			require.ErrorAs(t, err, &forbidden)
			require.Equal(t, rootfs.MustPath("/"+tc.file), forbidden.Rule())
			data, err := os.ReadFile(out)
			require.NoError(t, err)
			require.Equal(t, "original", string(data))
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1, "failed bundles must remove their temporary archive")
		})
	}
}

func TestBundleRejectsInvalidAccountJSONBeforeLoading(t *testing.T) {
	for _, flag := range []string{"--users", "--groups"} {
		for _, tc := range []struct{ name, value string }{
			{"empty input", ""},
			{"incomplete array", "["},
			{"object instead of array", `{"name":"app","gid":1000}`},
			{"invalid field type", `[{"gid":"invalid"}]`},
			{"trailing garbage", `[{"name":"app","gid":1000}]garbage`},
			{"second JSON value", `[{"name":"app","gid":1000}] []`},
		} {
			t.Run(flag+"/"+tc.name, func(t *testing.T) {
				fixture := newAppFixture(t)
				fixture.files["var/lib/dpkg/status"] = nil
				out := filepath.Join(t.TempDir(), "bundle.tar")
				_, err := captureStdout(t, func() error {
					return fixture.run("bundle", "--out", out, flag, tc.value)
				})
				require.ErrorContains(t, err, "for flag "+flag[1:]+":")
				require.NoFileExists(t, out)
			})
		}
	}
}

func TestBundleRejectsInvalidAccounts(t *testing.T) {
	for _, tc := range []struct{ name, users, groups, want string }{
		{"root group", "[]", `[{"name":"app","gid":0}]`, "GID or UID 0 is not allowed"},
		{"root user", `[{"name":"app","uid":0,"gid":1000}]`, `[{"name":"app","gid":1000}]`, "GID or UID 0 is not allowed"},
		{"root primary group", `[{"name":"app","uid":1000,"gid":0}]`, "[]", "GID or UID 0 is not allowed"},
		{"missing primary group", `[{"name":"app","uid":1000,"gid":1000}]`, "[]", "GID 1000 which is not defined in groups"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newAppFixture(t)
			out := filepath.Join(t.TempDir(), "bundle.tar")
			require.NoError(t, os.WriteFile(out, []byte("original"), 0644))
			err := fixture.run("bundle", "--out", out, "--users", tc.users, "--groups", tc.groups)
			require.ErrorContains(t, err, tc.want)
			data, err := os.ReadFile(out)
			require.NoError(t, err)
			require.Equal(t, "original", string(data))
		})
	}
}
