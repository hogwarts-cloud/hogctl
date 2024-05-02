package models

type Instance struct {
	Name     string   `yaml:"name"`
	Flavor   Flavor   `yaml:"flavor"`
	UserInfo UserInfo `yaml:"userInfo"`
}

type UserInfo struct {
	PublicKey string `yaml:"publicKey"`
}
