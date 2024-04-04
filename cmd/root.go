/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mhristof/go-aws-reachability/awscli"
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
		if len(args) != 2 {
			logger.Fatalf("Usage: %s <source> <dest>", os.Args[0])
		}

		a := awscli.NewAWS()

		sourceID, err := a.InstanceID(args[0])
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		destParts := strings.Split(args[1], ":")
		if len(destParts) != 2 {
			destParts = append(destParts, "80")
			logger.Warnf("No port specified, using 80")
		}

		destID, err := a.InstanceID(destParts[0])
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		logger.Infof("Checking reachability from %s to %s", sourceID, destID)

		portInt, err := strconv.Atoi(destParts[1])
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		reachable, err := a.Reachable(sourceID, destID, int32(portInt))
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		fmt.Println("reachable:", reachable)
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
}
