package handler

type Config struct {
	Port  int    `opts:"help=port, env"`
	User  string `opts:"help=default user when not provided in URL, env"`
	Token string `opts:"help=github api token, env=GH_TOKEN"`
}

var DefaultConfig = Config{
	Port: 3000,
	User: "jpillora",
}
