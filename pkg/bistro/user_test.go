package bistro

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserGroupNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"app", true}, {"a", true}, {"_", true}, {"_app-1", true},
		{"nobody", true}, {"nogroup", true},
		{"", false}, {"root", false}, {"1app", false}, {"App", false},
		{"app user", false}, {"app:admin", false}, {"app\nadmin", false},
		{"../app", false},
	} {
		t.Run(fmt.Sprintf("%q", tc.name), func(t *testing.T) {
			for kind, account := range map[string]Validator{
				"user":  User{Name: tc.name, Uid: 1000, Gid: 1000},
				"group": Group{Name: tc.name, Gid: 1000},
			} {
				t.Run(kind, func(t *testing.T) {
					if tc.valid {
						require.NoError(t, account.Validate())
					} else {
						require.Error(t, account.Validate())
					}
				})
			}
		})
	}
}

func TestUserGroupIDBoundaries(t *testing.T) {
	for _, id := range []uint16{0, 1, 65533, 65534, 65535} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			for kind, account := range map[string]Validator{
				"user uid":  User{Name: "app", Uid: id, Gid: 1000},
				"user gid":  User{Name: "app", Uid: 1000, Gid: id},
				"group gid": Group{Name: "app", Gid: id},
			} {
				t.Run(kind, func(t *testing.T) {
					if id == 0 || id == 65535 {
						require.EqualError(t, account.Validate(), fmt.Sprintf("GID or UID %d is not allowed", id))
					} else {
						require.NoError(t, account.Validate())
					}
				})
			}
		})
	}
}
