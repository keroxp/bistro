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
	if g.Name == "root" {
		return fmt.Errorf("Group name cannot be 'root'")
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
	if u.Name == "root" {
		return fmt.Errorf("User name cannot be 'root'")
	}
	if !validUserGroupPattern.MatchString(u.Name) {
		return fmt.Errorf("Invalid user name: %s", u.Name)
	}
	return nil
}

var validUserGroupPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateUGId(id uint16) error {
	if id == 0 || id == 0xffff {
		return fmt.Errorf("GID or UID %d is not allowed", id)
	}
	return nil
}
