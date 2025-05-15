package config

type Config interface {
	Protocol() string
	Server() string
	ReadBufferSize() int

	Auth() AuthConfig
}

type AuthConfig interface {
	User() string
	Pass() string
}
