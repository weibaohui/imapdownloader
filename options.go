package main

type Options struct {
	Dir      string   `yaml:"dir"`
	Host     string   `yaml:"host"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Prefixes []string `yaml:"prefixes"`
}
