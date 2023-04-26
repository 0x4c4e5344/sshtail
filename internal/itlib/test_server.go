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

package itlib

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/drognisep/sshtail/internal/testcontainers/openssh_server"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
	"os"
	"path"
	"testing"
)

func startTestContainer(t *testing.T, ctx context.Context, conf openssh_server.OpensshServerConfig) *openssh_server.OpensshServerContainer {
	server, err := openssh_server.NewOpensshServerContainer(ctx, conf)
	require.NoError(t, err, "failed to create openssh server container")
	return server
}

func writeEd2559PrivateKeyToDisk(privateKey ed25519.PrivateKey, identityFile string) error {
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ed25519 private key to PKCS8: %w", err)
	}

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	pemByte := pem.EncodeToMemory(block)

	err = os.WriteFile(identityFile, pemByte, 0600)
	if err != nil {
		return fmt.Errorf("failed to write ed25519 private key to disk: %w", err)
	}

	return nil
}

func writeEd2559PublicKeyToDisk(publicKey ed25519.PublicKey, identityFile string) error {
	publicKeyBytes, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create ssh public key from ed25519 public key: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes.Marshal(),
	}

	pemByte := pem.EncodeToMemory(block)

	pubKeyFile := identityFile + ".pub"
	err = os.WriteFile(pubKeyFile, pemByte, 0644)
	if err != nil {
		return fmt.Errorf("failed to write ed25519 public key to disk: %w", err)
	}

	return nil
}

type TestServer struct {
	config     openssh_server.OpensshServerConfig
	container  *openssh_server.OpensshServerContainer
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey

	IdentityFile string
	Username     string
	Hostname     string
	Port         int
}

func StartTestServer(t *testing.T, ctx context.Context) *TestServer {
	publicKey, privateKey, err := openssh_server.GenerateEd25519KeyPair()
	require.NoError(t, err, "failed to generate ed25519 key pair")

	identityFile := path.Join(t.TempDir() + "/id_ed25519")
	err = writeEd2559PrivateKeyToDisk(privateKey, identityFile)
	require.NoError(t, err, "failed to write ed25519 private key to disk")
	err = writeEd2559PublicKeyToDisk(publicKey, identityFile)
	require.NoError(t, err, "failed to write ed25519 public key to disk")

	serverConf := openssh_server.OpensshServerConfig{
		LoginName:       "testuser",
		LoginPassword:   "testpass",
		RequirePassword: false,
		PublicKey:       publicKey,
		PrivateKey:      privateKey,
	}

	server := startTestContainer(t, ctx, serverConf)

	return &TestServer{
		config:     serverConf,
		container:  server,
		publicKey:  publicKey,
		privateKey: privateKey,

		IdentityFile: identityFile,
		Username:     serverConf.LoginName,
		Hostname:     server.Hostname,
		Port:         server.Port,
	}
}

func StopTestServer(t *testing.T, ctx context.Context, server *TestServer) {
	err := server.container.Terminate(ctx)
	require.NoError(t, err, "failed to stop openssh server container")
}
