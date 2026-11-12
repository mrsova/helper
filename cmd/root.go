package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "helper",
	Short: "Утилита для помощи",
	Long:  `Консольный помощник.`,
	Run: func(cmd *cobra.Command, args []string) {
		runMenu()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
