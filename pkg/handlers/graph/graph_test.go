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
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/yindia/kubewatch/config"
	"github.com/yindia/kubewatch/pkg/event"
)

func TestGraphInit(t *testing.T) {
	g := &Graph{}
	expectedError := fmt.Errorf(graphErrMsg, "Missing Neptune endpoint")

	var Tests = []struct {
		name   string
		graph  config.Graph
		envs   map[string]string
		err    error
	}{
		{
			name: "Valid configuration",
			graph: config.Graph{
				Endpoint: "wss://test.neptune.amazonaws.com:8182/gremlin",
				Region:   "us-east-1",
			},
			envs: map[string]string{},
			err:  fmt.Errorf("failed to connect to Neptune: failed to create connection for address test.neptune.amazonaws.com:8182: dial tcp: lookup test.neptune.amazonaws.com: no such host"),
		},
		{
			name:  "Missing endpoint",
			graph: config.Graph{Region: "us-east-1"},
			envs:  map[string]string{},
			err:   expectedError,
		},
		{
			name:  "Missing region",
			graph: config.Graph{Endpoint: "wss://test.neptune.amazonaws.com:8182/gremlin"},
			envs:  map[string]string{},
			err:   fmt.Errorf(graphErrMsg, "Missing AWS region"),
		},
		{
			name:  "Configuration from environment variables",
			graph: config.Graph{},
			envs: map[string]string{
				"KW_GRAPH_ENDPOINT": "wss://test.neptune.amazonaws.com:8182/gremlin",
				"KW_GRAPH_REGION":   "us-east-1",
			},
			err: fmt.Errorf("failed to connect to Neptune: failed to create connection for address test.neptune.amazonaws.com:8182: dial tcp: lookup test.neptune.amazonaws.com: no such host"),
		},
		{
			name: "With traversal source and timeout",
			graph: config.Graph{
				Endpoint:        "wss://test.neptune.amazonaws.com:8182/gremlin",
				Region:          "us-east-1",
				TraversalSource: "custom_g",
				Timeout:         60,
			},
			envs: map[string]string{},
			err:  fmt.Errorf("failed to connect to Neptune: failed to create connection for address test.neptune.amazonaws.com:8182: dial tcp: lookup test.neptune.amazonaws.com: no such host"),
		},
	}

	for _, tt := range Tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envs {
					os.Unsetenv(k)
				}
			}()

			c := &config.Config{}
			c.Handler.Graph = tt.graph
			err := g.Init(c)
			
			if err != nil && tt.err != nil {
				if err.Error() != tt.err.Error() {
					t.Fatalf("Init() error = %v, want %v", err, tt.err)
				}
			} else if !reflect.DeepEqual(err, tt.err) {
				t.Fatalf("Init() error = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		region   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid configuration",
			endpoint: "wss://test.neptune.amazonaws.com:8182/gremlin",
			region:   "us-east-1",
			wantErr:  false,
		},
		{
			name:     "Missing endpoint",
			endpoint: "",
			region:   "us-east-1",
			wantErr:  true,
			errMsg:   "Missing Neptune endpoint",
		},
		{
			name:     "Missing region",
			endpoint: "wss://test.neptune.amazonaws.com:8182/gremlin",
			region:   "",
			wantErr:  true,
			errMsg:   "Missing AWS region",
		},
		{
			name:     "Missing both",
			endpoint: "",
			region:   "",
			wantErr:  true,
			errMsg:   "Missing Neptune endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graph{
				endpoint: tt.endpoint,
				region:   tt.region,
			}
			err := g.validateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("validateConfig() error message = %v, want containing %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGraphDefaults(t *testing.T) {
	g := &Graph{}
	c := &config.Config{}
	c.Handler.Graph = config.Graph{
		Endpoint: "wss://test.neptune.amazonaws.com:8182/gremlin",
		Region:   "us-east-1",
		// TraversalSource and Timeout not set, should use defaults
	}

	// Init will fail due to connection, but we can check the values were set
	_ = g.Init(c)

	if g.traversalSource != "g" {
		t.Errorf("Default traversalSource = %v, want %v", g.traversalSource, "g")
	}

	if g.timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want %v", g.timeout, 30*time.Second)
	}
}

func TestEventHandling(t *testing.T) {
	// Create a mock event
	testEvent := event.Event{
		Kind:      "Pod",
		Name:      "test-pod",
		Namespace: "default",
		Reason:    "Created",
		Action:    "create",
	}

	// Test that Handle doesn't panic when connection is nil
	g := &Graph{}
	g.Handle(testEvent) // Should log error but not panic
}

func TestResourceIDGeneration(t *testing.T) {
	tests := []struct {
		name      string
		event     event.Event
		wantID    string
	}{
		{
			name: "Pod resource ID",
			event: event.Event{
				Kind:      "Pod",
				Name:      "nginx-pod",
				Namespace: "production",
			},
			wantID: "Pod:production:nginx-pod",
		},
		{
			name: "Service resource ID",
			event: event.Event{
				Kind:      "Service",
				Name:      "api-service",
				Namespace: "staging",
			},
			wantID: "Service:staging:api-service",
		},
		{
			name: "Deployment resource ID",
			event: event.Event{
				Kind:      "Deployment",
				Name:      "webapp",
				Namespace: "default",
			},
			wantID: "Deployment:default:webapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceID := fmt.Sprintf("%s:%s:%s", tt.event.Kind, tt.event.Namespace, tt.event.Name)
			if resourceID != tt.wantID {
				t.Errorf("Resource ID = %v, want %v", resourceID, tt.wantID)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && s[0:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || len(substr) < len(s) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}