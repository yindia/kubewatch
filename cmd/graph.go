/*
Copyright 2018 Bitnami

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"strconv"

	"github.com/bitnami-labs/kubewatch/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// graphConfigCmd represents the graph subcommand
var graphConfigCmd = &cobra.Command{
	Use:   "graph",
	Short: "specific graph configuration",
	Long:  `specific graph configuration for Amazon Neptune`,
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.New()
		if err != nil {
			logrus.Fatal(err)
		}

		endpoint, err := cmd.Flags().GetString("endpoint")
		if err == nil {
			if len(endpoint) > 0 {
				conf.Handler.Graph.Endpoint = endpoint
			}
		} else {
			logrus.Fatal(err)
		}

		region, err := cmd.Flags().GetString("region")
		if err == nil {
			if len(region) > 0 {
				conf.Handler.Graph.Region = region
			}
		} else {
			logrus.Fatal(err)
		}

		enabled, err := cmd.Flags().GetString("enabled")
		if err == nil {
			if len(enabled) > 0 {
				isEnabled, err := strconv.ParseBool(enabled)
				if err != nil {
					logrus.Fatal(err)
				}
				conf.Handler.Graph.Enabled = isEnabled
			}
		} else {
			logrus.Fatal(err)
		}

		traversalSource, err := cmd.Flags().GetString("traversal-source")
		if err == nil {
			if len(traversalSource) > 0 {
				conf.Handler.Graph.TraversalSource = traversalSource
			}
		} else {
			logrus.Fatal(err)
		}

		timeout, err := cmd.Flags().GetString("timeout")
		if err == nil {
			if len(timeout) > 0 {
				timeoutInt, err := strconv.Atoi(timeout)
				if err != nil {
					logrus.Fatal(err)
				}
				conf.Handler.Graph.Timeout = timeoutInt
			}
		} else {
			logrus.Fatal(err)
		}

		tlsSkip, err := cmd.Flags().GetString("tlsskip")
		if err == nil {
			if len(tlsSkip) > 0 {
				skip, err := strconv.ParseBool(tlsSkip)
				if err != nil {
					logrus.Fatal(err)
				}
				conf.Handler.Graph.TlsSkip = skip
			} else {
				conf.Handler.Graph.TlsSkip = false
			}
		} else {
			logrus.Fatal(err)
		}

		if err = conf.Write(); err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	graphConfigCmd.Flags().StringP("endpoint", "e", "", "Specify Neptune endpoint URL (e.g., wss://your-cluster.region.neptune.amazonaws.com:8182/gremlin)")
	graphConfigCmd.Flags().StringP("region", "r", "", "Specify AWS region where Neptune cluster is located")
	graphConfigCmd.Flags().StringP("enabled", "", "", "Enable graph handler; TRUE or FALSE")
	graphConfigCmd.Flags().StringP("traversal-source", "t", "", "Specify graph traversal source (default: g)")
	graphConfigCmd.Flags().StringP("timeout", "", "", "Specify connection timeout in seconds (default: 30)")
	graphConfigCmd.Flags().StringP("tlsskip", "", "", "Specify whether to skip TLS verification; TRUE or FALSE")
}
