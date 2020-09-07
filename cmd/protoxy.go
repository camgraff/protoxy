package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&protoPath, "proto", "p", "", "path to the .proto file which contains your message definitions")
	rootCmd.MarkPersistentFlagRequired("proto")
}

// Flags
var protoPath string

var rootCmd = cobra.Command{
	Use: "protoxy",
	Run: startCmdFunc,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
