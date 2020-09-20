package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&importPaths, "import-paths", "I", nil, "paths to search for imports declared in your proto files")
	// TODO: Consider moving these into functions args instead of flags
	rootCmd.PersistentFlags().StringSliceVarP(&protoFiles, "proto", "p", nil, "path to the .proto files which contains your message definitions")
	rootCmd.MarkPersistentFlagRequired("proto")
	rootCmd.PersistentFlags().Uint16Var(&port, "port", 7777, "the port to start the server on")
}

// Flags
var importPaths []string
var protoFiles []string
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
