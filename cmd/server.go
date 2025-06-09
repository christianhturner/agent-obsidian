/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/christianhturner/agent-obsidian/internal/daemon"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`Server provides commands for managing the MCP Server. You can use the following commands:

			agent-obsidian server -f --flags [command]

			-- Commands -- 
			start - starts the MCP server listening to stdin`)
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts MCP server on Stdin",
	Long:  `TODO:`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting background service")
		if err := daemon.StartDaemon(); err != nil {
			log.Fatal("failed to start server: %v", err)
		}
		// server := mcp.NewServer("agent-obsidian", "0.1.0")

		log.Println("Agent obsidian MCP Server starting...")
		// if err := server.Run(); err != nil {
		// 	log.Fatal(err)
		// }
		fmt.Println("server called")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Stopping server...")
		if err := daemon.StopDaemon(); err != nil {
			return fmt.Errorf("failed to stop server: %v", err)
		}
		fmt.Println("Server stopped successfully")
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Restarting server...")

		// Stop if running
		if daemon.IsRunning() {
			if err := daemon.StopDaemon(); err != nil {
				return fmt.Errorf("failed to stop server: %v", err)
			}
			fmt.Println("Server stopped")
		}

		// Start again
		if err := daemon.StartDaemon(); err != nil {
			return fmt.Errorf("failed to start server: %v", err)
		}
		fmt.Println("Server restarted successfully")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check server status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(daemon.GetStatus())
	},
}

// var runDaemonCmd = &cobra.Command{
// 	Use:    "run-daemon",
// 	Short:  "Run server in daemon mode (internal use)",
// 	Hidden: true,
// 	Run: func(cmd *cobra.Command, args []string) {
// 		srv := mcp.NewServer()
// 		srv.Start()
// 	},
// }

func init() {
	serverCmd.AddCommand(startCmd)
	serverCmd.AddCommand(stopCmd)
	serverCmd.AddCommand(restartCmd)
	serverCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
