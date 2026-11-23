package bistro

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/loilo-inc/bistro/pkg/rootfs"
	"github.com/stretchr/testify/require"
)

func TestBakerCreatesAccounts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		users         []User
		groups        []Group
		passwd, group string
	}{
		{
			name:   "groups only",
			groups: []Group{{Name: "app", Gid: 1000}},
			group:  "root:x:0:\napp:x:1000:\n",
		},
		{
			name:   "nobody with nogroup",
			users:  []User{{Name: "nobody", Uid: 65534, Gid: 65534}},
			groups: []Group{{Name: "nogroup", Gid: 65534}},
			passwd: "root:x:0:0:root:/root:/sbin/nologin\nnobody:x:65534:65534::/home/nobody:/sbin/nologin\n",
			group:  "root:x:0:\nnogroup:x:65534:\n",
		},
		{
			name:   "users sharing a primary group",
			users:  []User{{Name: "a", Uid: 1, Gid: 1000}, {Name: "worker", Uid: 1001, Gid: 1000}},
			groups: []Group{{Name: "app", Gid: 1000}},
			passwd: "root:x:0:0:root:/root:/sbin/nologin\na:x:1:1000::/home/a:/sbin/nologin\nworker:x:1001:1000::/home/worker:/sbin/nologin\n",
			group:  "root:x:0:\napp:x:1000:\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "rootfs.tar")
			err := NewBaker(DI{FS: fstest.MapFS{}}).CreateRootFS(&BakeInput{
				Out: rootfs.MustPath(out), Users: tc.users, Groups: tc.groups,
			})
			require.NoError(t, err)
			entries := readBundleTar(t, out)
			want := map[string]string{"etc/group": tc.group}
			if tc.passwd != "" {
				want["etc/passwd"] = tc.passwd
			}
			require.Len(t, entries, len(want))
			for name, body := range want {
				require.Equal(t, body, entries[name].body)
				require.Equal(t, int64(0644), entries[name].header.Mode)
				require.Zero(t, entries[name].header.Uid)
				require.Zero(t, entries[name].header.Gid)
			}
		})
	}
}

func TestBakerInvalidAccountsPreserveOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		users  []User
		groups []Group
		want   string
	}{
		{"root user name", []User{{Name: "root", Uid: 1000, Gid: 1000}}, []Group{{Name: "app", Gid: 1000}}, "User name cannot be 'root'"},
		{"root group name", nil, []Group{{Name: "root", Gid: 1000}}, "Group name cannot be 'root'"},
		{"missing primary group", []User{{Name: "app", Uid: 1000, Gid: 1000}}, nil, "GID 1000 which is not defined in groups"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "rootfs.tar")
			require.NoError(t, os.WriteFile(out, []byte("original"), 0644))
			err := NewBaker(DI{FS: fstest.MapFS{}}).CreateRootFS(&BakeInput{
				Out: rootfs.MustPath(out), Users: tc.users, Groups: tc.groups,
			})
			require.ErrorContains(t, err, tc.want)
			data, err := os.ReadFile(out)
			require.NoError(t, err)
			require.Equal(t, "original", string(data))
			files, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, files, 1, "failed bundles must remove temporary archives")
		})
	}
}

func TestBakerAccountWriteErrors(t *testing.T) {
	failure := errors.New("archive write failed")
	b := &baker{}
	for name, write := range map[string]func(rootfs.Archiver) error{
		"passwd": func(a rootfs.Archiver) error {
			return b.writeUsers(a, []User{{Name: "app", Uid: 1000, Gid: 1000}})
		},
		"group": func(a rootfs.Archiver) error {
			return b.writeGroups(a, []Group{{Name: "app", Gid: 1000}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			a := &failingMetadataRoot{failAt: 1, err: failure}
			require.ErrorIs(t, write(a), failure)
		})
	}
}
