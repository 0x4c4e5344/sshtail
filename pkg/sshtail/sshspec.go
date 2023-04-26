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
	"context"
	"errors"
	"fmt"
	"github.com/drognisep/sshtail/pkg/specfile"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func noOpBanner(_ string) error { return nil }

// LoadKey reads a key from file
func LoadKey(path string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		_, ok := err.(*ssh.PassphraseMissingError)
		if !ok {
			return nil, err
		}

		fmt.Printf("Key %s requires a passphrase\n", path)
		fmt.Printf("Enter passphrase: ")
		passwd, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %v", err)
		}

		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passwd)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key")
		}

		fmt.Println("Key decrypted")
	}

	return ssh.PublicKeys(signer), nil
}

// ClientFilePair associates a Client connection with a host tag and file
type ClientFilePair struct {
	Client  *ssh.Client
	HostTag string
	File    string
}

func NewClientFilePair(hostTag string, host *specfile.HostSpec, key *specfile.KeySpec) (*ClientFilePair, error) {
	authMethod, err := LoadKey(key.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load key from %s: %w", key.Path, err)
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

	clientPair := &ClientFilePair{
		Client:  client,
		HostTag: hostTag,
		File:    host.File,
	}
	return clientPair, nil
}

// setupClients validates the spec data and sets up ClientFilePair instances.
func setupClients(specData *specfile.SpecData) ([]*ClientFilePair, error) {
	var err error
	clientPairs := make([]*ClientFilePair, 0, len(specData.Hosts))

	err = specData.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid spec data: %v", err)
	}

	for hostTag, v := range specData.Hosts {
		clientPair, err := NewClientFilePair(hostTag, v, specData.Keys[hostTag])
		if err != nil {
			return nil, err
		}

		clientPairs = append(clientPairs, clientPair)
	}

	return clientPairs, nil
}

// TailChannelWriter is a wrapper around a channel that implements the io.Writer interface.
type TailChannelWriter struct {
	prefix string
	ch     chan<- string
}

func (t TailChannelWriter) Write(b []byte) (int, error) {
	t.ch <- fmt.Sprintf("%s | %s", t.prefix, string(b))
	return len(b), nil
}

// TailSession represents a single tail session.
type TailSession struct {
	client  *ClientFilePair
	session *ssh.Session
	closed  bool
	wg      *sync.WaitGroup
}

// Closed returns whether the tail session has been previously closed. A closed tail session cannot be restarted.
func (s *TailSession) Closed() bool {
	return s.closed
}

// Started returns whether the tail session has already been started.
func (s *TailSession) Started() bool {
	return s.session != nil
}

// Close stops the running tail session and disconnects the client.
func (s *TailSession) Close() error {
	if s.closed {
		return nil
	}

	_, _ = fmt.Fprintf(os.Stderr, "Closing session to %s\n", s.client.HostTag)

	errorString := strings.Builder{}
	e1 := s.session.Close()
	if e1 != nil {
		errorString.WriteString(e1.Error())
	}

	e2 := s.client.Client.Close()
	if e2 != nil {
		errorString.WriteString(e2.Error())
	}

	if errorString.Len() > 0 {
		return fmt.Errorf("error(s) closing tail session: %s", errorString.String())
	}

	s.closed = true
	s.wg.Done()

	return nil
}

// Start the tail session using configured parameters
func (s *TailSession) Start(ch chan<- string, wg *sync.WaitGroup) error {
	if s.session != nil {
		return errors.New("tail session is already started")
	}

	var err error
	s.session, err = s.client.Client.NewSession()
	if err != nil {
		return fmt.Errorf("error establishing session: %v", err)
	}

	s.session.Stdout = TailChannelWriter{s.client.HostTag, ch}
	go func() {
		wg.Add(1)
		s.wg = wg
		cmd := fmt.Sprintf("tail -f %s", s.client.File)

		// We don't care if tail exits abruptly, ignoring the error
		_ = s.session.Run(cmd)
	}()

	return nil
}

// NewTailSession creates a new TailSession instance that is ready to be started.
func NewTailSession(client *ClientFilePair) *TailSession {
	return &TailSession{client, nil, false, nil}
}

// ConsolidatedWriter receives messages from all of its tail session instances and writes them to its output stream.
type ConsolidatedWriter struct {
	ch          chan string
	sessions    []*TailSession
	out         io.Writer
	started     bool
	closed      bool
	outputFiles []*os.File
}

// NewConsolidatedWriter creates tail sessions that are ready to Start and write to the provided writer.
func NewConsolidatedWriter(specData *specfile.SpecData, out io.Writer) (*ConsolidatedWriter, error) {
	clientPairs, err := setupClients(specData)
	if err != nil {
		return nil, err
	}

	numHosts := len(specData.Hosts)
	var ch = make(chan string, 1024*numHosts)
	sessions := make([]*TailSession, 0, numHosts)
	for _, pair := range clientPairs {
		ts := NewTailSession(pair)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, ts)
	}

	writer := &ConsolidatedWriter{ch, sessions, out, false, false, []*os.File{}}
	return writer, nil
}

// AddOutputFile adds a file to the list of files that should have output appended to them.
func (c *ConsolidatedWriter) AddOutputFile(file *os.File) error {
	_, err := file.Stat()
	if err != nil {
		return err
	}

	c.outputFiles = append(c.outputFiles, file)

	return nil
}

// Close closes all tail sessions as well as the connected clients.
func (c *ConsolidatedWriter) Close() error {
	if c.closed {
		return nil
	}

	for _, ts := range c.sessions {
		if !ts.Started() || ts.Closed() {
			continue
		}

		_ = ts.Close()
	}

	if len(c.outputFiles) > 0 {
		for _, f := range c.outputFiles {
			_ = f.Close()
		}
	}

	c.closed = true

	return nil
}

// Start starts all tail sessions. In the event of an error, all already opened sessions are closed and an error is returned.
func (c *ConsolidatedWriter) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, ts := range c.sessions {
		if ts.Started() || ts.Closed() {
			continue
		}

		err := ts.Start(c.ch, &wg)
		if err != nil {
			_ = c.Close()
			return err
		}
	}

	_, _ = fmt.Fprintf(os.Stderr, "Started tailing, send interrupt signal to exit\n")
	go func(ctx context.Context) {
		for {
			select {
			case line, ok := <-c.ch:
				if !ok {
					_ = c.Close()
					return
				}

				// Write to output buffer
				_, _ = c.out.Write([]byte(line))

				// Write to output files
				for _, o := range c.outputFiles {
					_, err := o.WriteString(line)
					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "[ERROR] Failed to write line to '%s'\n", o.Name())
					}
				}
			case <-ctx.Done():
				_ = c.Close()
				return
			}
		}
	}(ctx)

	wg.Wait()
	_, _ = fmt.Fprintln(os.Stderr, "Shutdown complete")

	return nil
}
