package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "compare [flags] [command]",
		Short: "compare extraction result",
		Args:  cobra.NoArgs,
	}

	// Add sub command
	rootCmd.AddCommand(cmdContent(), cmdAuthor())

	// Execute
	err := rootCmd.Execute()
	if err != nil {
		logrus.Fatalln(err)
	}
}

func cmdContent() *cobra.Command {
	return &cobra.Command{
		Use:   "content",
		Short: "compare accuracy for content extraction",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			compareContentExtraction()
		},
	}
}

func cmdAuthor() *cobra.Command {
	return &cobra.Command{
		Use:   "author",
		Short: "check accuracy for author extraction",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			compareAuthorExtraction()
		},
	}
}
