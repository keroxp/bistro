package app

import (
	"os"
	"path/filepath"
	"testing"

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
			out := filepath.Join(t.TempDir(), "bundle.tar")
			require.NoError(t, fixture.run("bundle", "--out", out,
				"--users", tc.users, "--groups", tc.groups))
			require.Equal(t, map[string]string{
				"etc/passwd": "root:x:0:0:root:/root:/sbin/nologin\n" + tc.passwd,
				"etc/group":  "root:x:0:\n" + tc.group,
			}, readArchiveFiles(t, out))
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
		{"root group", "[]", `[{"name":"app","gid":0}]`, "GID 0 is not allowed"},
		{"root user", `[{"name":"app","uid":0,"gid":1000}]`, `[{"name":"app","gid":1000}]`, "GID 0 or UID 0"},
		{"root primary group", `[{"name":"app","uid":1000,"gid":0}]`, "[]", "GID 0 or UID 0"},
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
