package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/friedenberg/lux/internal/capabilities"
	"github.com/friedenberg/lux/internal/config"
	"github.com/friedenberg/lux/internal/control"
	"github.com/friedenberg/lux/internal/server"
)

var rootCmd = &cobra.Command{
	Use:   "lux",
	Short: "Lux: LSP Multiplexer",
	Long:  `Lux multiplexes LSP requests to multiple language servers based on file type.`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the LSP server",
	Long:  `Start the Lux LSP server, reading from stdin and writing to stdout.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		srv, err := server.New(cfg)
		if err != nil {
			return fmt.Errorf("creating server: %w", err)
		}

		return srv.Run(cmd.Context())
	},
}

var addCmd = &cobra.Command{
	Use:   "add <flake>",
	Short: "Add an LSP from a nix flake",
	Long:  `Add a new LSP to the configuration by bootstrapping it to discover capabilities.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flake := args[0]
		return capabilities.Bootstrap(cmd.Context(), flake)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured LSPs",
	Long:  `List all LSPs configured in the Lux configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.LSPs) == 0 {
			fmt.Println("No LSPs configured")
			return nil
		}

		for _, lsp := range cfg.LSPs {
			fmt.Printf("%-20s %s\n", lsp.Name, lsp.Flake)
			if len(lsp.Extensions) > 0 {
				fmt.Printf("  extensions: %v\n", lsp.Extensions)
			}
			if len(lsp.Patterns) > 0 {
				fmt.Printf("  patterns:   %v\n", lsp.Patterns)
			}
			if len(lsp.LanguageIDs) > 0 {
				fmt.Printf("  languages:  %v\n", lsp.LanguageIDs)
			}
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of running LSPs",
	Long:  `Connect to a running Lux server and show the status of all LSPs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Status(os.Stdout)
	},
}

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Eagerly start an LSP",
	Long:  `Start a configured LSP without waiting for a matching request.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Start(args[0])
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running LSP",
	Long:  `Stop a running LSP to free resources.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		client, err := control.NewClient(cfg.SocketPath())
		if err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer client.Close()

		return client.Stop(args[0])
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
