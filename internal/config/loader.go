package config

type Config struct {
	protocol string
	server   string

	auth *AuthConfig
}

func (c *Config) Protocol() string {
	return c.protocol
}

func (c *Config) Server() string {
	return c.server
}

type AuthConfig struct {
	user string
	pass string
}

func (c *Config) User() string {
	return c.auth.user
}
func (c *Config) Pass() string {
	return c.auth.pass
}

func LoadDefaultConfig() *Config {
	return &Config{
		protocol: "tcp",
		server:   "localhost:9090",

		auth: &AuthConfig{
			user: "fran150",
			pass: "123456",
		},
	}
}
