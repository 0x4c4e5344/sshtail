/*
 * Copyright (c) 2020 Joseph Saylor <doug@saylorsolutions.com>
 * Copyright (c) 2023 Lorenzo Delgado <lnsdev@proton.me>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package openssh_server

import (
	"context"
	"fmt"
	"github.com/drognisep/sshtail/internal/projutils"
	wait2 "github.com/drognisep/sshtail/internal/testcontainers/wait"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"path"
	"strconv"
	"time"
)

type OpensshServerContainer struct {
	testcontainers.Container
	config   OpensshServerConfig
	Hostname string
	Port     int
}

func NewOpensshServerContainer(ctx context.Context, config OpensshServerConfig) (*OpensshServerContainer, error) {
	projRoot, err := projutils.RootDir()
	if err != nil {
		return nil, fmt.Errorf("couldn't get project root path: %w", err)
	}

	buildContextPath := path.Join(projRoot, "test", "testdata")

	sshSigner, err := config.GetSshSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh signer from ed25519 private key: %w", err)
	}

	sshPublicKey, err := config.GetSshPublicKeyAsString()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh public key from ed25519 public key: %w", err)
	}
	request := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{Context: buildContextPath},
		ExposedPorts:   []string{"2222/tcp"},
		Env: map[string]string{
			"PUID":            "1000",
			"PGID":            "1000",
			"TZ":              "Etc/UTC",
			"PASSWORD_ACCESS": strconv.FormatBool(config.RequirePassword),
			"USER_NAME":       config.LoginName,
			"USER_PASSWORD":   config.LoginPassword,
			"PUBLIC_KEY":      sshPublicKey,
		},
		WaitingFor: wait.ForAll(
			wait.NewHostPortStrategy("2222/tcp"),
			wait2.NewSshProbeStrategy(sshSigner, config.LoginName).WithPort("2222/tcp"),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
		Logger:           testcontainers.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	sshPort, err := container.MappedPort(ctx, "2222/tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to get container ssh port: %w", err)
	}

	opensshServerContainer := &OpensshServerContainer{
		config:    config,
		Container: container,
		Hostname:  host,
		Port:      sshPort.Int(),
	}
	return opensshServerContainer, nil
}
