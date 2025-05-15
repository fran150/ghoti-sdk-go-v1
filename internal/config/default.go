package config

type DefaultConfig struct {
	protocol string
	server   string

	readBufferSize int

	auth AuthConfig
}

func (c *DefaultConfig) Protocol() string {
	return c.protocol
}

func (c *DefaultConfig) Server() string {
	return c.server
}

func (c *DefaultConfig) Auth() AuthConfig {
	return c.auth
}

func (c *DefaultConfig) ReadBufferSize() int {
	return c.readBufferSize
}

type DefaultAuthConfig struct {
	user string
	pass string
}

func (a *DefaultAuthConfig) User() string {
	return a.user
}
func (a *DefaultAuthConfig) Pass() string {
	return a.pass
}

func LoadDefaultConfig() Config {
	return &DefaultConfig{
		protocol: "tcp",
		server:   "localhost:9090",

		readBufferSize: (8 * 1024),

		auth: &DefaultAuthConfig{
			user: "test_a_service",
			pass: "67890",
		},
	}
}
