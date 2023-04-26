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
	"fmt"
	"github.com/docker/go-connections/nat"
	"golang.org/x/crypto/ssh"
	"net"
)

const ProbeCommand = "exit 0"

type SshProbe struct {
	sshConfig ssh.ClientConfig
	host      string
	port      string
}

func NewSshProbe(signer ssh.Signer, loginName string) SshProbe {
	sshConfig := ssh.ClientConfig{
		User:            loginName,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return SshProbe{sshConfig, "localhost", "22"}
}

func (s SshProbe) WithHost(host string) SshProbe {
	s.host = host
	return s
}

func (s SshProbe) WithPort(port nat.Port) SshProbe {
	s.port = port.Port()
	return s
}

func (s SshProbe) Run() error {
	address := net.JoinHostPort(s.host, s.port)

	client, err := ssh.Dial("tcp", address, &s.sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh session creation failed: %w", err)
	}
	defer session.Close()

	if err = session.Run(ProbeCommand); err != nil {
		return err
	}

	return nil
}
