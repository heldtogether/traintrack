package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var setBackend = &cobra.Command{
	Use:   "set-instance <url>",
	Short: "Configure the CLI to use a specific instance",
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		fmt.Printf("Setting Traintrack instance to %s\n", url)
		RunSetBackend(url)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires exactly one argument: the URL to your instance")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setBackend)
}

func RunSetBackend(url string) {
	conf := &InstanceConfig{
		URL: url,
	}
	conf = conf.refreshAuthConfig()
	SaveConfig(DefaultConfigPath, conf)
}
