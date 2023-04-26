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

package sshtail

import (
	"errors"
	"fmt"
	"github.com/drognisep/sshtail/pkg/specfile"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
)

// TailChannelWriter is a wrapper around a channel that implements the io.Writer interface.
type TailChannelWriter struct {
	prefix string
	ch     chan<- string
}

func (t TailChannelWriter) Write(b []byte) (int, error) {
	t.ch <- fmt.Sprintf("%s | %s", t.prefix, string(b))
	return len(b), nil
}

// TailSshClient associates a client connection with a host tag and spec data.
type TailSshClient struct {
	tag     string
	host    *specfile.HostSpec
	client  *ssh.Client
	session *ssh.Session
}

func noOpBanner(_ string) error { return nil }

func NewTailSshClient(hostTag string, host *specfile.HostSpec) (*TailSshClient, error) {
	authMethod, err := LoadKey(host.IdentityFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load key from %s: %w", host.IdentityFile, err)
	}

	addr := net.JoinHostPort(host.Hostname, strconv.Itoa(host.Port))
	config := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		BannerCallback:  noOpBanner,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	clientPair := &TailSshClient{
		client: client,
		tag:    hostTag,
		host:   host,
	}
	return clientPair, nil
}

// Started returns true if the client has an active session.
func (c *TailSshClient) Started() bool {
	return c.session != nil
}

// StartSession starts a new tail session on the client.
func (c *TailSshClient) StartSession(ch chan<- string) error {
	if c.session != nil {
		return errors.New("session already started")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	session.Stdout = TailChannelWriter{c.tag, ch}

	cmd := fmt.Sprintf("tail -f %s", c.host.File)
	err = session.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to execute tail session command: %w", err)
	}

	c.session = session

	return nil
}

// Close closes the client connection and the session.
func (c *TailSshClient) Close() error {
	if c.session != nil {
		_ = c.session.Signal(ssh.SIGINT)
		_ = c.session.Close()
		c.session = nil
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %v", err)
	}

	return nil
}
