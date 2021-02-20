package main

import "github.com/spf13/cobra"

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tipocket",
		Short: "TiPocket toolset",
	}
	rootCmd.AddCommand(newInitCmd())
	rootCmd.Execute()
}
