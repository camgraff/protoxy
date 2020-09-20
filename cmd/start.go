package cmd

import (
	"fmt"

	"github.com/camgraff/protoxy/protoparser"
	"github.com/camgraff/protoxy/server"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server",
	RunE:  startCmdFunc,
}

func startCmdFunc(command *cobra.Command, args []string) error {
	fd, err := protoparser.FileDescriptorsFromPaths(importPaths, protoFiles)
	if err != nil {
		return fmt.Errorf("Invalid proto path: %w", err)
	}
	cfg := server.Config{
		FileDescriptors: fd,
		Port:            port,
	}
	srv := server.New(cfg)
	srv.Run()
	return nil
}
