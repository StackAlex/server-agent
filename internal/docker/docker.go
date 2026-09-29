package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
	State  string `json:"state"`
}

func GetContainers(projectDir string) ([]Container, error) {
	cmd := exec.Command(
		"docker",
		"ps",
		"-a",
		"--format",
		"{{json .}}",
	)

	cmd.Dir = projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(
			"docker ps failed: %w",
			err,
		)
	}

	var containers []Container

	scanner := bufio.NewScanner(
		bytes.NewReader(output),
	)

	for scanner.Scan() {
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var raw struct {
			ID     string `json:"ID"`
			Names  string `json:"Names"`
			Image  string `json:"Image"`
			Status string `json:"Status"`
			State  string `json:"State"`
		}

		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf(
				"parse docker output: %w",
				err,
			)
		}

		containers = append(
			containers,
			Container{
				ID:     raw.ID,
				Name:   raw.Names,
				Image:  raw.Image,
				Status: raw.Status,
				State:  raw.State,
			},
		)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return containers, nil
}
