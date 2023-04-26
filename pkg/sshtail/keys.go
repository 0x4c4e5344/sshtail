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
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"syscall"
)

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
