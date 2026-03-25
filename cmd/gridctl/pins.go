package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gridctl/gridctl/pkg/pins"
	"github.com/gridctl/gridctl/pkg/state"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const pinsAPITimeout = 10 * time.Second

var pinsStack string
var pinsExitCode bool

var pinsCmd = &cobra.Command{
	Use:   "pins",
	Short: "Manage schema pins for MCP servers",
	Long:  "Inspect, verify, approve, and reset TOFU schema pins that protect against rug pull attacks.",
}

var pinsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pin status for all servers in a stack",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPinsList()
	},
}

var pinsVerifyCmd = &cobra.Command{
	Use:   "verify [server]",
	Short: "Show verification status for servers",
	Long:  "Show pin verification status. Pass --exit-code to exit 1 on drift (for CI).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		server := ""
		if len(args) == 1 {
			server = args[0]
		}
		return runPinsVerify(server)
	},
}

var pinsApproveCmd = &cobra.Command{
	Use:   "approve <server>",
	Short: "Approve schema changes for a server",
	Long:  "Re-pin the current tool definitions for a server, clearing drift status.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPinsApprove(args[0])
	},
}

var pinsResetCmd = &cobra.Command{
	Use:   "reset <server>",
	Short: "Delete pins for a server",
	Long:  "Remove all pins for a server. It will be re-pinned on next deploy.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPinsReset(args[0])
	},
}

func init() {
	pinsCmd.PersistentFlags().StringVar(&pinsStack, "stack", "", "Stack name (auto-detected if only one stack is deployed)")
	pinsVerifyCmd.Flags().BoolVar(&pinsExitCode, "exit-code", false, "Exit with code 1 if drift is detected (for CI)")

	pinsCmd.AddCommand(pinsListCmd)
	pinsCmd.AddCommand(pinsVerifyCmd)
	pinsCmd.AddCommand(pinsApproveCmd)
	pinsCmd.AddCommand(pinsResetCmd)
}

// resolveStack returns the stack name, auto-detecting when only one stack is deployed.
func resolveStack() (string, error) {
	if pinsStack != "" {
		return pinsStack, nil
	}

	states, err := state.List()
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("listing stacks: %w", err)
	}

	switch len(states) {
	case 0:
		return "", fmt.Errorf("no running stack found. Deploy a stack first")
	case 1:
		return states[0].StackName, nil
	default:
		names := make([]string, len(states))
		for i, s := range states {
			names[i] = s.StackName
		}
		return "", fmt.Errorf("multiple stacks found %v. Use --stack to specify one", names)
	}
}

// resolveRunningStack returns the daemon state for the active stack.
func resolveRunningStack() (*state.DaemonState, error) {
	stackName, err := resolveStack()
	if err != nil {
		return nil, err
	}

	st, err := state.Load(stackName)
	if err != nil {
		return nil, fmt.Errorf("stack '%s' not found", stackName)
	}

	if !state.IsRunning(st) {
		return nil, fmt.Errorf("stack '%s' is not running. Deploy the stack first", st.StackName)
	}

	return st, nil
}

func runPinsList() error {
	stackName, err := resolveStack()
	if err != nil {
		return err
	}

	ps := pins.New(stackName)
	if err := ps.Load(); err != nil {
		return err
	}

	servers := ps.GetAll()
	if len(servers) == 0 {
		fmt.Printf("No pins found for stack '%s'. Deploy the stack first.\n", stackName)
		return nil
	}

	names := sortedMapKeys(servers)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"SERVER", "TOOLS", "STATUS", "LAST VERIFIED"})

	for _, name := range names {
		sp := servers[name]
		t.AppendRow(table.Row{
			name,
			sp.ToolCount,
			pinStatusLabel(sp.Status),
			sp.LastVerifiedAt.Format("2006-01-02 15:04:05"),
		})
	}

	t.Render()
	return nil
}

func runPinsVerify(server string) error {
	stackName, err := resolveStack()
	if err != nil {
		return err
	}

	ps := pins.New(stackName)
	if err := ps.Load(); err != nil {
		return err
	}

	servers := ps.GetAll()
	if len(servers) == 0 {
		fmt.Printf("No pins found for stack '%s'. Deploy the stack first.\n", stackName)
		return nil
	}

	if server != "" {
		sp, ok := servers[server]
		if !ok {
			return fmt.Errorf("no pins found for server %q. Deploy the stack first", server)
		}
		servers = map[string]*pins.ServerPins{server: sp}
	}

	names := sortedMapKeys(servers)
	hasDrift := false

	for _, name := range names {
		sp := servers[name]
		switch sp.Status {
		case pins.StatusDrift:
			hasDrift = true
			fmt.Printf("  ✗ %-24s drift detected\n", name)
		case pins.StatusApprovedPendingRedeploy:
			fmt.Printf("  ~ %-24s %d tools approved, pending redeploy\n", name, sp.ToolCount)
		default:
			fmt.Printf("  ✓ %-24s %d tools verified\n", name, sp.ToolCount)
		}
	}

	if pinsExitCode && hasDrift {
		os.Exit(1)
	}
	return nil
}

func runPinsApprove(server string) error {
	st, err := resolveRunningStack()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%d/api/pins/%s/approve", st.Port, server)
	client := &http.Client{Timeout: pinsAPITimeout}

	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("calling pins API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no pins found for server %q. Deploy the stack first", server)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("approve failed: %s", string(body))
	}

	var result struct {
		ToolCount int    `json:"tool_count"`
		Server    string `json:"server"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Printf("✓ Approved schema update for %s (%d tools re-pinned)\n", server, result.ToolCount)
	return nil
}

func runPinsReset(server string) error {
	st, err := resolveRunningStack()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://localhost:%d/api/pins/%s", st.Port, server)
	client := &http.Client{Timeout: pinsAPITimeout}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling pins API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no pins found for server %q. Deploy the stack first", server)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("reset failed: %s", string(body))
	}

	fmt.Printf("✓ Pins reset for %s. Server will be re-pinned on next deploy.\n", server)
	return nil
}

// pinStatusLabel returns a human-readable display string for a pin status.
func pinStatusLabel(status string) string {
	switch status {
	case pins.StatusPinned:
		return "✓ pinned"
	case pins.StatusDrift:
		return "⚠ drift"
	case pins.StatusApprovedPendingRedeploy:
		return "~ approved"
	default:
		return "— " + status
	}
}

// sortedMapKeys returns sorted keys from a string-keyed map.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
