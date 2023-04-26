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

package wait

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
	"time"
)

type SshProbeStrategy struct {
	timeout      *time.Duration
	Port         nat.Port
	testCmd      SshProbe
	PollInterval time.Duration
}

func NewSshProbeStrategy(signer ssh.Signer, loginName string) *SshProbeStrategy {
	return &SshProbeStrategy{
		Port:         "22/tcp",
		testCmd:      NewSshProbe(signer, loginName),
		PollInterval: 200 * time.Millisecond,
	}
}

// WithStartupTimeout can be used to change the default startup timeout
func (s *SshProbeStrategy) WithStartupTimeout(startupTimeout time.Duration) *SshProbeStrategy {
	s.timeout = &startupTimeout
	return s
}

// WithPort can be used to override the default port
func (s *SshProbeStrategy) WithPort(port nat.Port) *SshProbeStrategy {
	s.Port = port
	return s
}

// WithPollInterval can be used to override the default polling interval of 100 milliseconds
func (s *SshProbeStrategy) WithPollInterval(pollInterval time.Duration) *SshProbeStrategy {
	s.PollInterval = pollInterval
	return s
}

func (s *SshProbeStrategy) Timeout() *time.Duration {
	return s.timeout
}

func (s *SshProbeStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	timeout := 60 * time.Second // default testcontainers container startup timeout
	if s.timeout != nil {
		timeout = *s.timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ipAddress, err := target.Host(ctx)
	if err != nil {
		return err
	}

	var port nat.Port
	port, err = target.MappedPort(ctx, s.Port)

	for port == "" {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s:%w", ctx.Err(), err)
		case <-time.After(s.PollInterval):
			port, err = target.MappedPort(ctx, s.Port)
			if err != nil {
				return err
			}
		}
	}

	if port.Proto() != "tcp" {
		return errors.New("cannot use SSH client on non-TCP ports")
	}

	sshCmd := s.testCmd.WithHost(ipAddress).WithPort(port)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.PollInterval):
			var exitCode int
			var exitError *ssh.ExitError

			err = sshCmd.Run()
			if err == nil {
				exitCode = 0
			} else if errors.As(err, &exitError) {
				exitCode = err.(*ssh.ExitError).ExitStatus()
			} else {
				return err
			}

			if exitCode != 0 {
				continue
			}

			return nil
		}
	}
}
