package main

import (
	"os"

	"github.com/jannfis/argocd-agent/cmd/cmd"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewServerRunCommand() *cobra.Command {
	var (
		listenHost string
		listenPort int
		logLevel   string
	)
	var command = &cobra.Command{
		Short: "Run the argocd-agent server component",
		Run: func(c *cobra.Command, args []string) {
			if logLevel != "" {
				_, err := cmd.StringToLoglevel(logLevel)
				if err != nil {
					cmd.Fatal("invalid log level: %s. Available levels are: %s", logLevel, cmd.AvailableLogLevels())
				}
			}
		},
	}
	command.Flags().StringVar(&listenHost, "listen-host", "", "Name of the host to listen on")
	command.Flags().IntVar(&listenPort, "listen-port", 8443, "Port to listen on")
	command.Flags().StringVar(&logLevel, "log-level", logrus.InfoLevel.String(), "The log level to use")
	return command
}

func main() {
	cmd := NewServerRunCommand()
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
