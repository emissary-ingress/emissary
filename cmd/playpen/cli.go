package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/datawire/teleproxy/pkg/supervisor"
	"github.com/spf13/cobra"
)

func (d *Daemon) handleCommand(p *supervisor.Process, conn net.Conn, data *ClientMessage) error {
	out := NewEmitter(conn)
	rootCmd := &cobra.Command{
		Use:          "playpen",
		SilenceUsage: true, // https://github.com/spf13/cobra/issues/340
		RunE: func(_ *cobra.Command, _ []string) error {
			out.Println("Running \"playpen status\". Use \"playpen help\" to get help.")
			if err := d.Status(p, out); err != nil {
				return err
			}
			return out.Err()
		},
	}
	rootCmd.SetOutput(conn) // FIXME replace with SetOut and SetErr
	rootCmd.SetArgs(data.Args[1:])

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "show program's version number and exit",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			out.Println("Client", data.ClientVersion)
			out.Println("Daemon", displayVersion)
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:    "daemon-foreground",
		Short:  "launch Playpen Daemon in the foreground (debug)",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			out.Println("Daemon", displayVersion, "is already running.")
			out.Println("Use \"playpen quit\" to terminate the daemon.")
			out.SendExit(1)
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "daemon",
		Short: "launch Playpen Daemon in the background (sudo)",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			out.Println("Daemon", displayVersion, "is already running.")
			out.Println("Use \"playpen quit\" to terminate the daemon.")
			out.SendExit(1)
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "show connectivity status",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := d.Status(p, out); err != nil {
				return err
			}
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "connect [-- additional kubectl arguments...]",
		Short: "connect to a cluster",
		RunE: func(_ *cobra.Command, args []string) error {
			d.Connect(p, out, data.RAI, args)
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "disconnect",
		Short: "disconnect from the connected cluster",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := d.Disconnect(p, out); err != nil {
				return err
			}
			return out.Err()
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "quit",
		Short: "tell Playpen Daemon to quit (for upgrades)",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			out.Println("Playpen Daemon quitting...")
			p.Supervisor().Shutdown()
			return out.Err()
		},
	})

	interceptCmd := &cobra.Command{
		Use:   "intercept",
		Long:  "Manage deployment intercepts. An intercept arranges for a subset of requests to be diverted to the local machine.",
		Short: "manage deployment intercepts",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := d.ListIntercepts(p, out); err != nil {
				return err
			}
			return out.Err()
		},
	}
	interceptCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "list current intercepts",
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := d.ListIntercepts(p, out); err != nil {
				return err
			}
			return out.Err()
		},
	})
	interceptCmd.AddCommand(&cobra.Command{
		Use:   "remove",
		Short: "deactivate and remove an existent intercept",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if err := d.RemoveIntercept(p, out, name); err != nil {
				return err
			}
			return out.Err()
		},
	})
	intercept := InterceptInfo{}
	interceptAddCmd := &cobra.Command{
		Use:   "add DEPLOYMENT -t [HOST:]PORT -m HEADER=REGEX ...",
		Short: "add a deployment intercept",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			intercept.Deployment = args[0]
			if intercept.Name == "" {
				intercept.Name = fmt.Sprintf("cept-%d", time.Now().Unix())
			}

			var host, portStr string
			hp := strings.SplitN(intercept.TargetHost, ":", 2)
			if len(hp) < 2 {
				portStr = hp[0]
			} else {
				host = strings.TrimSpace(hp[0])
				portStr = hp[1]
			}
			if len(host) == 0 {
				host = "127.0.0.1"
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				out.Printf("Failed to parse %q as HOST:PORT: %v", intercept.TargetHost, err)
				out.SendExit(1)
				return nil
			}
			intercept.TargetHost = host
			intercept.TargetPort = port
			if err := d.AddIntercept(p, out, &intercept); err != nil {
				return err
			}
			return out.Err()
		},
	}
	interceptAddCmd.Flags().StringVarP(&intercept.Name, "name", "n", "", "a name for this intercept")
	interceptAddCmd.Flags().StringVarP(&intercept.TargetHost, "target", "t", "", "the [HOST:]PORT to forward to")
	interceptAddCmd.MarkFlagRequired("target")
	interceptAddCmd.Flags().StringToStringVarP(&intercept.Patterns, "match", "m", nil, "match expression (HEADER=REGEX)")
	interceptAddCmd.MarkFlagRequired("match")

	interceptCmd.AddCommand(interceptAddCmd)
	rootCmd.AddCommand(interceptCmd)

	return rootCmd.Execute()
}
