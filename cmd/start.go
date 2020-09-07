package cmd

import (
	"github.com/camgraff/protoxy/server"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server",
	Run:   startCmdFunc,
}

func startCmdFunc(command *cobra.Command, args []string) {
	cfg := server.Config{
		ProtoPath: protoPath,
	}
	srv := server.New(cfg)
	srv.Run()
}
