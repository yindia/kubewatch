/*
Copyright 2016 Skippbox, Ltd.

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

package client

import (
	"net/http"
	"os"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yindia/kubewatch/config"
	"github.com/yindia/kubewatch/pkg/controller"
	"github.com/yindia/kubewatch/pkg/handlers"
	"github.com/yindia/kubewatch/pkg/handlers/graph"
	"github.com/sirupsen/logrus"
)

// Run runs the event loop processing with given handler
func Run(conf *config.Config) {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = ":2112"
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logrus.Infof("Starting metrics server on port %s", listenAddress)
		if err := http.ListenAndServe(listenAddress, nil); err != nil {
			logrus.Errorf("Error starting metrics server on port %s: %v", listenAddress, err)
		}
	}()

	var eventHandler = ParseEventHandler(conf)
	controller.Start(conf, eventHandler)
}

// ParseEventHandler returns the respective handler object specified in the config file.
func ParseEventHandler(conf *config.Config) handlers.Handler {

	var eventHandler handlers.Handler
	switch {
	case conf.Handler.Graph.Enabled && len(conf.Handler.Graph.Endpoint) > 0:
		eventHandler = new(graph.Graph)
	default:
		eventHandler = new(handlers.Default)
	}
	if err := eventHandler.Init(conf); err != nil {
		logrus.Fatal(err)
	}
	return eventHandler
}
