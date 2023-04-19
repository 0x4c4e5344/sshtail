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

package specfile

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
)

func noOpBanner(message string) error { return nil }

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
		passwd, err := terminal.ReadPassword(int(syscall.Stdin))
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

func createKnownHostsCallback() (ssh.HostKeyCallback, error) {
	c, _ := user.Current()
	knownHostPath := path.Join(c.HomeDir, ".ssh", "known_hosts")
	knownHostsCallback, err := knownhosts.New(knownHostPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create host key verification callback using '%s': %v", knownHostPath, err)
	}

	return knownHostsCallback, nil
}

// ClientFilePair associates a Client connection with a host tag and file
type ClientFilePair struct {
	Client  *ssh.Client
	HostTag string
	File    string
}

// setupClients validates the spec data and sets up ClientFilePair instances.
func setupClients(specData *SpecData) ([]*ClientFilePair, error) {
	var err error
	clientPairs := make([]*ClientFilePair, 0, len(specData.Hosts))

	err = specData.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid spec data: %v", err)
	}

	knownHostsCallback, err := createKnownHostsCallback()
	if err != nil {
		return nil, err
	}

	for k, v := range specData.Hosts {
		authMethod, err := LoadKey(specData.Keys[k].Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load key from %s: %v", specData.Keys[k].Path, err)
		}

		config := &ssh.ClientConfig{
			User: v.Username,
			Auth: []ssh.AuthMethod{
				authMethod,
			},
			BannerCallback:  noOpBanner,
			HostKeyCallback: knownHostsCallback,
		}
		config.SetDefaults()
		hostPort := fmt.Sprintf("%s:%d", v.Hostname, v.Port)
		client, err := ssh.Dial("tcp", hostPort, config)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %v", hostPort, err)
		}

		clientPairs = append(clientPairs, &ClientFilePair{client, k, v.File})
	}

	return clientPairs, nil
}

type TailChannelWriter struct {
	prefix string
	ch     chan<- string
}

func (t TailChannelWriter) Write(b []byte) (n int, err error) {
	t.ch <- fmt.Sprintf("[ %s ] %s", t.prefix, string(b))
	n = len(b)
	return
}

// TailSession represents
type TailSession struct {
	clientPair *ClientFilePair
	session    *ssh.Session
	closed     bool
	started    bool
	wg         *sync.WaitGroup
}

// Closed returns whether the tail session has been previously closed. A closed tail session cannot be restarted.
func (s *TailSession) Closed() bool {
	return s.closed
}

// Started returns whether the tail session has already been started.
func (s *TailSession) Started() bool {
	return s.started
}

// Close stops the running tail session and disconnects the client.
func (s *TailSession) Close() error {
	var err error
	if !s.closed {
		fmt.Printf("Closing session to %s\n", s.clientPair.HostTag)

		s.closed = true
		sb := strings.Builder{}
		errorsOccurred := false

		e1 := s.session.Close()
		if e1 != nil {
			sb.WriteString(e1.Error())
			errorsOccurred = true
		}

		e2 := s.clientPair.Client.Close()
		if e2 != nil {
			sb.WriteString(e2.Error())
			errorsOccurred = true
		}

		if errorsOccurred {
			err = fmt.Errorf("error(s) closing tail session: %s", sb.String())
		}

		s.wg.Done()
	}

	return err
}

// Start the tail session using configured parameters
func (s *TailSession) start(ch chan<- string, wg *sync.WaitGroup) error {
	if s.closed {
		return errors.New("can't start a closed tail session")
	}

	if s.started {
		return errors.New("tail session is already started")
	}

	session, err := s.clientPair.Client.NewSession()
	if err != nil {
		return fmt.Errorf("error establishing session: %v", err)
	}

	s.session = session
	session.Stdout = TailChannelWriter{s.clientPair.HostTag, ch}
	go func() {
		wg.Add(1)
		s.wg = wg
		cmd := fmt.Sprintf("tail -n 0 -f %s", s.clientPair.File)

		// I don't care that tail will exit ungracefully, not handling or reporting error
		_ = session.Run(cmd)
	}()
	s.started = true

	return nil
}

// NewTailSession creates a new TailSession instance that is ready to be started.
func NewTailSession(client *ClientFilePair) *TailSession {
	return &TailSession{client, nil, false, false, nil}
}

// ConsolidatedWriter receives messages from all of its tail session instances and writes them to its output stream.
type ConsolidatedWriter struct {
	ch          chan string
	sessions    []*TailSession
	out         *os.File
	started     bool
	closed      bool
	outputFiles []*os.File
}

// NewConsolidatedWriter creates tail sessions that are ready to start and write to the provided writer.
func NewConsolidatedWriter(specData *SpecData, out *os.File) (*ConsolidatedWriter, error) {
	clientPairs, err := setupClients(specData)
	if err != nil {
		return nil, err
	}

	numHosts := len(specData.Hosts)
	var ch = make(chan string, numHosts)
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
	for _, ts := range c.sessions {
		if ts.Started() && !ts.Closed() {
			_ = ts.Close()
		}
	}

	if len(c.outputFiles) > 0 {
		for _, f := range c.outputFiles {
			_ = f.Close()
		}
	}

	return nil
}

// Start starts all tail sessions. In the event of an error, all already opened sessions are closed and an error is returned.
func (c *ConsolidatedWriter) Start() error {
	var wg sync.WaitGroup
	for _, ts := range c.sessions {
		if !ts.Started() && !ts.Closed() {
			err := ts.start(c.ch, &wg)
			if err != nil {
				fmt.Println("Failed to start consolidated writer. Closing sessions.")
				_ = c.Close()
				return err
			}
		}
	}

	fmt.Printf("Started tailing, send interrupt signal to exit\n\n")
	go func() {
		for line := range c.ch {
			_, _ = c.out.WriteString(line)
			for _, o := range c.outputFiles {
				_, err := o.WriteString(line)
				if err != nil {
					fmt.Printf("[ERROR] Failed to write line to '%s'\n", o.Name())
				}
			}
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nSignal received, closing sessions")
		_ = c.Close()
	}()

	wg.Wait()
	fmt.Println("Shut down complete")
	return nil
}
