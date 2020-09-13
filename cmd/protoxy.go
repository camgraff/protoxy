package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&protoPath, "proto", "p", "", "path to the .proto file which contains your message definitions")
	rootCmd.MarkPersistentFlagRequired("proto")
	rootCmd.PersistentFlags().Uint16Var(&port, "port", 7777, "the port to start the server on")
}

// Flags
var protoPath string
var port uint16

var rootCmd = cobra.Command{
	Use:  "protoxy",
	RunE: startCmdFunc,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
