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
	"github.com/drognisep/sshtail/pkg/specfile"
	"io"
)

// setupClients validates the spec data and sets up TailSshClient instances.
func setupClients(specData *specfile.SpecData) ([]*TailSshClient, error) {
	clients := make([]*TailSshClient, 0, len(specData.Hosts))

	for tag, host := range specData.Hosts {
		client, err := NewTailSshClient(tag, host)
		if err != nil {
			return nil, err
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// ConsolidatedWriter receives messages from all of its tail session instances and writes them to its output stream.
type ConsolidatedWriter struct {
	ch      chan string
	clients []*TailSshClient
	out     io.Writer
}

// NewConsolidatedWriter creates tail sessions that are ready to Start and write to the provided writer.
func NewConsolidatedWriter(specData *specfile.SpecData, output io.Writer) (*ConsolidatedWriter, error) {
	clients, err := setupClients(specData)
	if err != nil {
		return nil, err
	}

	writer := &ConsolidatedWriter{nil, clients, output}
	return writer, nil
}

// Close closes all tail sessions as well as the connected clients.
func (c *ConsolidatedWriter) Close() error {
	if c.ch == nil {
		return nil
	}

	for _, ts := range c.clients {
		if !ts.Started() {
			continue
		}

		_ = ts.Close()
	}

	close(c.ch)
	c.ch = nil

	return nil
}

// Start starts all tail sessions. In the event of an error, all already opened sessions are closed and an error is returned.
func (c *ConsolidatedWriter) Start(ctx context.Context) error {
	if c.ch != nil {
		return errors.New("already started")
	}

	c.ch = make(chan string, 1024*len(c.clients))
	for _, client := range c.clients {
		if client.Started() {
			continue
		}

		err := client.StartSession(c.ch)
		if err != nil {
			_ = c.Close()
			return err
		}
	}

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
			case <-ctx.Done():
				_ = c.Close()
				return
			}
		}
	}(ctx)

	return nil
}
