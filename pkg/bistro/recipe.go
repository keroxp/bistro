package bistro

import (
	"os"

	"go.yaml.in/yaml/v3"
)

type Recipe struct {
	Build  BuildConfig  `yaml:"build"`
	Bundle BundleConfig `yaml:"bundle"`
}

type BuildConfig struct {
	From       string      `yaml:"from"`
	User       string      `yaml:"user"`
	StopSignal string      `yaml:"stopsignal"`
	Workdir    string      `yaml:"workdir"`
	Volumes    []string    `yaml:"volumes"`
	Cmd        []string    `yaml:"cmd"`
	Entrypoint []string    `yaml:"entrypoint"`
	Labels     []NameValue `yaml:"labels"`
	Envs       []NameValue `yaml:"envs"`
}

type NameValue struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Group struct {
	Name string `yaml:"name"`
	Gid  int    `yaml:"gid"`
}

type User struct {
	Name string `yaml:"name"`
	Uid  int    `yaml:"uid"`
	Gid  int    `yaml:"gid"`
}

type BundleConfig struct {
	Files        []string `yaml:"files"`
	Groups       []Group  `yaml:"groups"`
	Users        []User   `yaml:"users"`
	Packages     []string `yaml:"packages"`
	Excludes     []string `yaml:"excludes"`
	ExcludeScans []string `yaml:"exclude_scans"`
}

func ReadRecipe(path string) (*Recipe, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var recipe Recipe
	if err := yaml.NewDecoder(file).Decode(&recipe); err != nil {
		return nil, err
	}
	return &recipe, nil
}
