/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/mhristof/go-aws-reachability/reach"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-aws-reachability",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose {
			logger.SetLevel(logger.DebugLevel)
		}

		logger.Infof("Args %v", args)
		if len(args) != 2 {
			logger.Fatalf("Usage: %s <source> <dest>", os.Args[0])
		}

		this := reach.NewTarget(args[0])
		logger.Infof("Checking reachability from %s to %s", this.InstanceID, args[1])

		target := args[1]
		destParts := strings.Split(args[1], ":")
		if len(destParts) != 2 {
			destParts = append(destParts, "80")
			logger.Warnf("No port specified, using 80")
		}

		portInt, err := strconv.Atoi(destParts[1])
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		reachable := this.CanReach(target, int32(portInt))
		logger.Infof("reachable: %v", reachable)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-aws-reachability.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// add verbose flag
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
