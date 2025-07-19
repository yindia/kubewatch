/*
Copyright 2025 Kubewatch Contributors

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

package graph

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/sirupsen/logrus"

	"github.com/yindia/kubewatch/config"
	"github.com/yindia/kubewatch/pkg/event"
)

var graphErrMsg = `
%s

You need to set Neptune endpoint and region for the Graph handler,
using environment variables:

export KW_GRAPH_ENDPOINT=wss://your-cluster.region.neptune.amazonaws.com:8182/gremlin
export KW_GRAPH_REGION=us-east-1

Or configure in the kubewatch config file.
`

// Graph handler implements handler.Handler interface,
// stores events as nodes and relationships in Amazon Neptune graph database
type Graph struct {
	endpoint        string
	region          string
	traversalSource string
	timeout         time.Duration
	tlsSkip         bool
	connection      *gremlingo.DriverRemoteConnection
	g               *gremlingo.GraphTraversalSource
}

// Init prepares Graph handler configuration
func (g *Graph) Init(c *config.Config) error {
	endpoint := c.Handler.Graph.Endpoint
	region := c.Handler.Graph.Region
	traversalSource := c.Handler.Graph.TraversalSource
	timeout := c.Handler.Graph.Timeout
	tlsSkip := c.Handler.Graph.TlsSkip

	// Check environment variables
	if endpoint == "" {
		endpoint = os.Getenv("KW_GRAPH_ENDPOINT")
	}
	if region == "" {
		region = os.Getenv("KW_GRAPH_REGION")
	}

	// Set defaults
	if traversalSource == "" {
		traversalSource = "g"
	}
	if timeout == 0 {
		timeout = 30 // Default 30 seconds
	}

	g.endpoint = endpoint
	g.region = region
	g.traversalSource = traversalSource
	g.timeout = time.Duration(timeout) * time.Second
	g.tlsSkip = tlsSkip

	// Validate required fields
	if err := g.validateConfig(); err != nil {
		return err
	}

	// Initialize Neptune connection
	return g.connect()
}

// validateConfig validates the Graph handler configuration
func (g *Graph) validateConfig() error {
	if g.endpoint == "" {
		return fmt.Errorf(graphErrMsg, "Missing Neptune endpoint")
	}
	if g.region == "" {
		return fmt.Errorf(graphErrMsg, "Missing AWS region")
	}
	return nil
}

// connect establishes connection to Neptune
func (g *Graph) connect() error {
	// Create connection settings
	settings := func(settings *gremlingo.DriverRemoteConnectionSettings) {
		settings.TraversalSource = g.traversalSource
		if g.tlsSkip {
			settings.TlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	// Create the connection
	connection, err := gremlingo.NewDriverRemoteConnection(g.endpoint, settings)
	if err != nil {
		return fmt.Errorf("failed to connect to Neptune: %v", err)
	}

	g.connection = connection
	g.g = gremlingo.Traversal_().WithRemote(connection)

	logrus.Printf("Successfully connected to Neptune at %s", g.endpoint)
	return nil
}

// Handle handles an event by creating nodes and relationships in the graph
func (g *Graph) Handle(e event.Event) {
	if g.g == nil {
		logrus.Error("Graph handler not properly initialized")
		return
	}

	// Create resource node
	err := g.createResourceNode(e)
	if err != nil {
		logrus.Errorf("Failed to create resource node: %v", err)
		return
	}

	// Create event node
	err = g.createEventNode(e)
	if err != nil {
		logrus.Errorf("Failed to create event node: %v", err)
		return
	}

	// Create relationship between resource and event
	err = g.createEventRelationship(e)
	if err != nil {
		logrus.Errorf("Failed to create event relationship: %v", err)
		return
	}

	logrus.Printf("Successfully stored event in graph: %s/%s", e.Namespace, e.Name)
}

// createResourceNode creates or updates a Kubernetes resource node
func (g *Graph) createResourceNode(e event.Event) error {
	resourceID := fmt.Sprintf("%s:%s:%s", e.Kind, e.Namespace, e.Name)
	
	// Check if node exists, create if not
	exists, err := g.g.V().HasId(resourceID).HasNext()
	if err != nil {
		return err
	}

	if !exists {
		// Create new resource node
		_, err = g.g.AddV("Resource").
			Property("id", resourceID).
			Property("kind", e.Kind).
			Property("name", e.Name).
			Property("namespace", e.Namespace).
			Property("createdAt", time.Now().Unix()).
			Property("lastUpdated", time.Now().Unix()).
			Next()
		if err != nil {
			return err
		}
	} else {
		// Update existing node
		_, err = g.g.V(resourceID).
			Property("lastUpdated", time.Now().Unix()).
			Next()
		if err != nil {
			return err
		}
	}

	return nil
}

// createEventNode creates an event node
func (g *Graph) createEventNode(e event.Event) error {
	eventID := fmt.Sprintf("event:%s:%d", e.Name, time.Now().UnixNano())
	
	_, err := g.g.AddV("Event").
		Property("id", eventID).
		Property("kind", e.Kind).
		Property("name", e.Name).
		Property("namespace", e.Namespace).
		Property("reason", e.Reason).
		Property("message", e.Message()).
		Property("timestamp", time.Now().Unix()).
		Next()
	
	return err
}

// createEventRelationship creates an edge between resource and event
func (g *Graph) createEventRelationship(e event.Event) error {
	resourceID := fmt.Sprintf("%s:%s:%s", e.Kind, e.Namespace, e.Name)
	eventID := fmt.Sprintf("event:%s:%d", e.Name, time.Now().UnixNano())
	
	// Create edge from resource to event
	_, err := g.g.V(resourceID).
		AddE("HAS_EVENT").
		To(gremlingo.T__.V(eventID)).
		Property("timestamp", time.Now().Unix()).
		Next()
	
	return err
}

// Close closes the Neptune connection
func (g *Graph) Close() error {
	if g.connection != nil {
		g.connection.Close()
	}
	return nil
}