package docker

import (
	"context"
	"fmt"

	"github.com/gridctl/gridctl/pkg/dockerclient"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// EnsureNetwork creates the network if it doesn't exist.
// The stack parameter is used for labeling (for cleanup).
func EnsureNetwork(ctx context.Context, cli dockerclient.DockerClient, name, driver, stack string) (string, error) {
	// Check if network exists
	networks, err := cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", name)),
	})
	if err != nil {
		return "", fmt.Errorf("listing networks: %w", err)
	}

	for _, n := range networks {
		if n.Name == name {
			return n.ID, nil
		}
	}

	// Create network with stack label for cleanup
	labels := map[string]string{
		LabelManaged: "true",
	}
	if stack != "" {
		labels[LabelStack] = stack
	}

	resp, err := cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: driver,
		Labels: labels,
	})
	if err != nil {
		return "", fmt.Errorf("creating network %s: %w", name, err)
	}

	return resp.ID, nil
}

// ListManagedNetworks returns all networks managed by gridctl for a stack.
func ListManagedNetworks(ctx context.Context, cli dockerclient.DockerClient, stack string) ([]string, error) {
	filterArgs := filters.NewArgs(
		filters.Arg("label", LabelManaged+"=true"),
	)
	if stack != "" {
		filterArgs.Add("label", LabelStack+"="+stack)
	}

	networks, err := cli.NetworkList(ctx, network.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}

	var names []string
	for _, n := range networks {
		names = append(names, n.Name)
	}
	return names, nil
}

// RemoveNetwork removes a network by name.
func RemoveNetwork(ctx context.Context, cli dockerclient.DockerClient, name string) error {
	return cli.NetworkRemove(ctx, name)
}
