package models

type Instance struct {
	Name      string    `yaml:"name"`
	Resources Resources `yaml:"resources"`
	User      User      `yaml:"user"`
}

type Resources struct {
	Flavor Flavor `yaml:"flavor"`
	Disk   int    `yaml:"disk"`
}

type User struct {
	Name      string `yaml:"name"`
	Email     string `yaml:"email"`
	PublicKey string `yaml:"publicKey"`
}
