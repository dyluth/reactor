package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/moby/term"
)

// AttachInteractiveSession attaches to a running container with TTY support
func (s *Service) AttachInteractiveSession(ctx context.Context, containerID string) error {
	// Check if container is running
	containerInfo, err := s.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	if !containerInfo.State.Running {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Set up TTY if we're in a terminal
	isTerminal := term.IsTerminal(os.Stdin.Fd())

	// Create exec instance for interactive shell
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          isTerminal,
		Cmd:          []string{"/bin/bash"}, // Default to bash, could be configurable
	}

	execResp, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Attach to the exec instance
	attachResp, err := s.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    isTerminal,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer attachResp.Close()

	// Handle TTY mode
	if isTerminal {
		// Save current terminal state
		oldState, err := term.SaveState(os.Stdin.Fd())
		if err != nil {
			return fmt.Errorf("failed to save terminal state: %w", err)
		}
		defer func() {
			if err := term.RestoreTerminal(os.Stdin.Fd(), oldState); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal state: %v\n", err)
			}
		}()

		// Set terminal to raw mode
		_, err = term.SetRawTerminal(os.Stdin.Fd())
		if err != nil {
			return fmt.Errorf("failed to set raw terminal: %w", err)
		}
	}

	// Copy stdin to container and container output to stdout/stderr
	errChan := make(chan error, 3)

	// Copy stdin to container
	go func() {
		_, err := io.Copy(attachResp.Conn, os.Stdin)
		errChan <- err
	}()

	// Copy container output to stdout
	go func() {
		_, err := io.Copy(os.Stdout, attachResp.Reader)
		errChan <- err
	}()

	// Wait for the session to end
	// This will block until the user exits the container session
	go func() {
		err := s.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{
			Detach: false,
			Tty:    isTerminal,
		})
		errChan <- err
	}()

	// Wait for first error or completion
	return <-errChan
}