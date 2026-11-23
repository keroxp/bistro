package bistro

import (
	"fmt"
	"regexp"
)

type Validator interface {
	Validate() error
}

type Group struct {
	Name string `yaml:"name" json:"name"`
	Gid  uint16 `yaml:"gid" json:"gid"`
}

func (g Group) Validate() error {
	if err := validateUGId(g.Gid); err != nil {
		return err
	}
	if !validUserGroupPattern.MatchString(g.Name) {
		return fmt.Errorf("Invalid group name: %s", g.Name)
	}
	return nil
}

type User struct {
	Name string `yaml:"name" json:"name"`
	Uid  uint16 `yaml:"uid" json:"uid"`
	Gid  uint16 `yaml:"gid" json:"gid"`
}

func (u User) Validate() error {
	if err := validateUGId(u.Uid); err != nil {
		return err
	}
	if err := validateUGId(u.Gid); err != nil {
		return err
	}
	if !validUserGroupPattern.MatchString(u.Name) {
		return fmt.Errorf("Invalid user name: %s", u.Name)
	}
	return nil
}

var validUserGroupPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateUGId(id uint16) error {
	if id == 0 {
		return fmt.Errorf("GID or UID 0 is not allowed")
	} else if id >= 65534 {
		return fmt.Errorf("GID or UID %d is out of valid range", id)
	}
	return nil
}
