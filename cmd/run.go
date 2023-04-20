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

package cmd

import (
	"fmt"
	"github.com/drognisep/sshtail/pkg/specfile"
	"github.com/drognisep/sshtail/pkg/sshtail"
	"os"

	"github.com/spf13/cobra"
)

var outputFiles []string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Args:  cobra.ExactArgs(1),
	Short: "Runs a spec file to connect to multiple hosts and tail the files specified",
	Long: `Spec files have the extension .spec. A template can be created with
	sshtail spec init your-spec-name-here`,
	RunE: func(cmd *cobra.Command, args []string) error {
		specData, err := specfile.ReadSpecFile(args[0])
		if err != nil {
			return fmt.Errorf("unable to parse config file '%s': %v", args[0], err)
		}

		writer, err := sshtail.NewConsolidatedWriter(specData, os.Stdout)
		if err != nil {
			return err
		}

		for _, s := range outputFiles {
			file, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open and append to file '%s'", s)
			}
			writer.AddOutputFile(file)
		}
		writer.Start()

		return nil
	},
}

func init() {
	specCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceVarP(&outputFiles, "output", "o", []string{}, "Adds a file to the list of files that should have messages appended")
}
