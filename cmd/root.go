/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/mhristof/go-aws-reachability/reach"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Version = "devel"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "reachability",
	Short:   "Create AWS Reachability insigts and runs them",
	Version: Version,
	Long: heredoc.Doc(`
		Reachability is a tool to check if an instance can reach another instance

		Examples:

		# check if instance1 can reach instance2
		$ reachability instance1 instance2

		# Check if instance1 can reach instance2 on port 8123
		$ reachability instance1 instance2:8123

		# Check if instance1 can reach an ecs service which has a route53 entry
		$ reachability instance1 service.ecs.local:8123
	`),
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose {
			logger.SetLevel(logger.DebugLevel)
		}

		logger.Debugf("Args %v", args)
		if len(args) != 2 {
			logger.Fatalf("Usage: %s <source> <dest>", os.Args[0])
		}

		this := reach.NewTarget(args[0])
		logger.Infof("Checking reachability from %s to %s", this.InstanceID, args[1])

		destParts := strings.Split(args[1], ":")
		if len(destParts) != 2 {
			destParts = append(destParts, "80")
			logger.Warnf("No port specified, using 80")
		}

		portInt, err := strconv.Atoi(destParts[1])
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		reachable := this.CanReach(destParts[0], int32(portInt))
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
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
