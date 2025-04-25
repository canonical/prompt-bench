package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func setupRootCmd() *cobra.Command {
	var count uint
	var parallel bool

	cmd := &cobra.Command{
		Use:   "prompting-bench FILE_OR_DIR",
		Short: "Permission prompting benchmarker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBench(args[0], count, parallel)
		},
	}

	cmd.Flags().UintVar(&count, "count", 1, "Number of times to run the request")
	cmd.Flags().BoolVar(&parallel, "parallel", false, "Run multiple requests in parallel")

	return cmd
}

func setupSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup DIRECTORY FILES_NUM [DIRS_NUM]",
		Short: "Create files and subdirectories in the destination folder",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			nFiles, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("the number of files to create is invalid: %v", err)
			}
			nDirectories := 0
			if len(args) > 2 {
				nDirectories, err = strconv.Atoi(args[2])
				if err != nil {
					return fmt.Errorf("the number of directories to create is invalid: %v", err)
				}
			}

			return setupFolder(args[0], nFiles, nDirectories)
		},
	}

	return cmd
}

func setupRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Print the number of rules currently managed by snapd and per snap",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printNumberOfRules()
		},
	}

	return cmd
}

func main() {
	rootCmd := setupRootCmd()
	rootCmd.AddCommand(setupSetupCmd())
	rootCmd.AddCommand(setupRulesCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
