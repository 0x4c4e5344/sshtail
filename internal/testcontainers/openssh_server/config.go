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
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

// GenerateEd25519KeyPair generates an ed25519 key pair.
func GenerateEd25519KeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	var pub ed25519.PublicKey
	var prv ed25519.PrivateKey

	pub, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return pub, prv, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	return pub, prv, nil
}

// OpensshServerConfig is a configuration for the OpenSSH server testcontainer.
type OpensshServerConfig struct {
	LoginName       string
	LoginPassword   string
	RequirePassword bool
	PublicKey       ed25519.PublicKey
	PrivateKey      ed25519.PrivateKey
}

// GetSshSigner returns an ssh.Signer for the private key.
func (c OpensshServerConfig) GetSshSigner() (ssh.Signer, error) {
	return ssh.NewSignerFromKey(c.PrivateKey)
}

// GetSshPublicKey returns an ssh.PublicKey for the public key.
func (c OpensshServerConfig) GetSshPublicKey() (ssh.PublicKey, error) {
	return ssh.NewPublicKey(c.PublicKey)
}

// GetSshPublicKeyAsString returns a string representation of the public key.
func (c OpensshServerConfig) GetSshPublicKeyAsString() (string, error) {
	pub, err := c.GetSshPublicKey()
	if err != nil {
		return "", err
	}

	return string(ssh.MarshalAuthorizedKey(pub)), nil
}
