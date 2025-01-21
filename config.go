package main

type RevolverPortConfig struct {
	Port int    `yaml:"port"`
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
}

type RevolverScriptConfig struct {
	Preload string `yaml:"preload"`
	Run     string `yaml:"run"`
	CleanUp string `yaml:"cleanup"`
}

type RevolverConfig struct {
	ProjectRootFolder       string               `yaml:"root"`
	ExecutablePackageFolder string               `yaml:"exec"`
	Ports                   []RevolverPortConfig `yaml:"ports"`
	Scripts                 RevolverScriptConfig `yaml:"scripts"`
	ObservingExts           []string             `yaml:"exts"`
}
