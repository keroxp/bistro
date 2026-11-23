package bistro

type Group struct {
	Name string `yaml:"name" json:"name"`
	Gid  int    `yaml:"gid" json:"gid"`
}

type User struct {
	Name string `yaml:"name" json:"name"`
	Uid  int    `yaml:"uid" json:"uid"`
	Gid  int    `yaml:"gid" json:"gid"`
}
