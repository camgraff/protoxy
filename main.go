package main

import "github.com/camgraff/proto-proxy/server"

func main() {
	srv := server.New()
	srv.Run()
}
