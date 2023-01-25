package handler

// Config installer handler
type Config struct {
	Host  string `opts:"help=host, env=HTTP_HOST"`
	Port  int    `opts:"help=port, env"`
	User  string `opts:"help=default user when not provided in URL, env"`
	Token string `opts:"help=github api token, env=GITHUB_TOKEN"`
}

// DefaultConfig for an installer handler
var DefaultConfig = Config{
	Port: 3000,
	User: "jpillora",
}
