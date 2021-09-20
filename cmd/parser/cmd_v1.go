package parser

type CmdV1 struct {
	Cmd  string `yaml:"cmd"`
	Ver  string `yaml:"ver"`
	Desc string `yaml:"desc"`
	Util []struct {
		Name string `yaml:"name"`
		Desc string `yaml:"desc"`
		Args string `yaml:"args"`
	} `yaml:"util"`
	Install []struct {
		Name       string `yaml:"name"`
		InstallCmd string `yaml:"install_cmd"`
	} `yaml:"install"`
	Docs []string `yaml:"docs"`
}
