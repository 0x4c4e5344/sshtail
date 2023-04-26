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

// Package specfile contains the logic for parsing the spec files.
package specfile

import (
	"errors"
	"fmt"
	"io"
	"os/user"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultSshPort int = 22

func defaultUsername() string {
	u, err := user.Current()
	if err != nil {
		// May need to check for sudo user on linux. Not going to support
		// edge cases like this initially.
		fmt.Println("Warning: Unable to determine current user")
	}

	split := strings.Split(u.Username, "\\")

	return split[len(split)-1]
}

func defaultIdentityFile() string {
	u, _ := user.Current()
	return path.Join(u.HomeDir, ".ssh", "id_rsa")
}

// HostSpec encapsulates the parameters for a single host to tail.
type HostSpec struct {
	Hostname     string `yaml:"hostname"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	IdentityFile string `yaml:"identity_file"`
	File         string `yaml:"file"`
}

// Validate checks the HostSpec for errors and sets reasonable defaults.
func (h *HostSpec) Validate() error {
	if h.Hostname == "" {
		return errors.New("cannot have a blank hostname")
	}

	if h.Port == 0 {
		h.Port = DefaultSshPort
	}

	if h.Username == "" {
		h.Username = defaultUsername()
	}

	if h.IdentityFile == "" {
		h.IdentityFile = defaultIdentityFile()
	}

	if h.File == "" {
		return errors.New("cannot have a blank file")
	}

	return nil
}

// SpecData encapsulates runtime parameters for SSH tailing.
type SpecData struct {
	Hosts map[string]*HostSpec `yaml:"hosts"`
}

// Validate checks the SpecData for errors and sets reasonable defaults.
func (s *SpecData) Validate() error {
	if s.Hosts == nil || len(s.Hosts) == 0 {
		return errors.New("hosts must have at least one definition")
	}

	for k, v := range s.Hosts {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("host spec %s: %w", k, err)
		}
	}

	return nil
}

// LoadSpecData reads the SpecData and validates it.
func LoadSpecData(reader io.Reader) (*SpecData, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("unable to read spec data: %w", err)
	}

	specData := &SpecData{}
	if err = yaml.Unmarshal(data, specData); err != nil {
		return nil, fmt.Errorf("invalid spec data format: %w", err)
	}

	if err = specData.Validate(); err != nil {
		return nil, fmt.Errorf("invalid spec data: %w", err)
	}

	return specData, nil
}
