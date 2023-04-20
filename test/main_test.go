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

package test

import (
	"bytes"
	"context"
	"github.com/drognisep/sshtail/internal/itlib"
	"github.com/drognisep/sshtail/pkg/specfile"
	"github.com/drognisep/sshtail/pkg/sshtail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	/// Setup
	ctx := context.Background()

	serverA := itlib.StartTestServer(t, ctx)
	defer itlib.StopTestServer(t, ctx, serverA)

	serverB := itlib.StartTestServer(t, ctx)
	defer itlib.StopTestServer(t, ctx, serverB)

	/// Given
	specData := &specfile.SpecData{
		Hosts: map[string]*specfile.HostSpec{
			"serverA": {
				Hostname: serverA.Hostname,
				Username: serverA.Username,
				File:     "/app/logs/test.log",
				Port:     serverA.Port,
			},
			"serverB": {
				Hostname: serverB.Hostname,
				Username: serverB.Username,
				File:     "/app/logs/test.log",
				Port:     serverB.Port,
			},
		},
		Keys: map[string]*specfile.KeySpec{
			"serverA": {
				Path: serverA.IdentityFile,
			},
			"serverB": {
				Path: serverB.IdentityFile,
			},
		},
	}

	buffer := bytes.Buffer{}

	/// When
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	writer, err := sshtail.NewConsolidatedWriter(specData, &buffer)
	require.NoError(t, err, "failed to create consolidated writer")

	go func() {
		err = writer.Start(ctx)
		require.NoErrorf(t, err, "failed to start consolidated writer: %v", err)
	}()

	time.Sleep(2 * time.Second)

	/// Then
	lines := strings.Split(buffer.String(), "\n")
	assert.Lenf(t, lines, 7, "expected 7 lines, got %d", len(lines))
}
