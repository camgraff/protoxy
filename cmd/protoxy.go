package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringSliceVarP(&importPaths, "import-paths", "I", nil, "paths to search for imports declared in your proto files. Defaults to current directory.")
	rootCmd.MarkPersistentFlagRequired("proto")
	rootCmd.PersistentFlags().Uint16Var(&port, "port", 7777, "the port to start the server on")
}

// Flags
var importPaths []string
var port uint16

var rootCmd = cobra.Command{
	Use:   "protoxy PROTO_FILES",
	Short: "Start the proxy server",
	Long: `Start a proxy server that converts JSON request bodies 
to Protocol Buffers. See github.com/camgraff/protoxy for documentation`,
	Args: cobra.MinimumNArgs(1),
	RunE: startCmdFunc,
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
