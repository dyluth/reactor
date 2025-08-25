package docker

import (
	"context"
	"fmt"
)

// ProvisionContainer implements the three-phase container recovery strategy:
// 1. Check for running container with deterministic name
// 2. Check for stopped container and restart it  
// 3. Create new container only if needed
func (s *Service) ProvisionContainer(ctx context.Context, spec *ContainerSpec) (ContainerInfo, error) {
	// Phase 1: Check if container already exists
	containerInfo, err := s.ContainerExists(ctx, spec.Name)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("failed to check container existence: %w", err)
	}

	switch containerInfo.Status {
	case StatusRunning:
		// Container is already running - return it
		return containerInfo, nil

	case StatusStopped:
		// Container exists but is stopped - restart it
		if err := s.StartContainer(ctx, containerInfo.ID); err != nil {
			// If restart fails, remove the broken container and create new one
			if removeErr := s.RemoveContainer(ctx, containerInfo.ID); removeErr != nil {
				return ContainerInfo{}, fmt.Errorf("failed to start container %s and failed to remove it: start error: %w, remove error: %v", containerInfo.ID, err, removeErr)
			}
			// Fall through to create new container
			break
		}
		
		// Successfully restarted
		containerInfo.Status = StatusRunning
		return containerInfo, nil

	case StatusNotFound:
		// Container doesn't exist - will create new one below
		break
	}

	// Phase 3: Create new container
	newContainer, err := s.CreateContainer(ctx, spec)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("failed to create new container: %w", err)
	}

	// Start the newly created container
	if err := s.StartContainer(ctx, newContainer.ID); err != nil {
		// Clean up failed container
		if removeErr := s.RemoveContainer(ctx, newContainer.ID); removeErr != nil {
			return ContainerInfo{}, fmt.Errorf("failed to start new container %s and failed to remove it: start error: %w, remove error: %v", newContainer.ID, err, removeErr)
		}
		return ContainerInfo{}, fmt.Errorf("failed to start new container: %w", err)
	}

	newContainer.Status = StatusRunning
	return newContainer, nil
}

