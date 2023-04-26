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
	"context"
	"fmt"
	"github.com/drognisep/sshtail/pkg/specfile"
	"github.com/drognisep/sshtail/pkg/sshtail"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Args:  cobra.ExactArgs(1),
	Short: "Runs a spec file to connect to multiple hosts and tail the files specified",
	Long: `Spec files have the extension .spec. A template can be created with
	sshtail spec init your-spec-name-here`,
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("failed to open config file '%s': %w", args[0], err)
		}

		specData, err := specfile.LoadSpecData(specFile)
		if err != nil {
			return fmt.Errorf("unable to parse config file '%s': %w", args[0], err)
		}

		writer, err := sshtail.NewConsolidatedWriter(specData, os.Stdout)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			_, _ = fmt.Fprintln(os.Stderr, "Signal received, closing sessions")
			_ = writer.Close()
		}()

		_, _ = fmt.Fprintf(os.Stderr, "Started tailing, send interrupt signal to exit\n")
		if err = writer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start: %w", err)
		}

		return nil
	},
}

func init() {
	specCmd.AddCommand(runCmd)
}
