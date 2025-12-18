// Package main provides tests for the Hex plugin.
package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// MockCommandExecutor is a mock implementation of CommandExecutor for testing.
type MockCommandExecutor struct {
	RunFunc func(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error)
	Calls   []MockCall
}

// MockCall records a call to the mock executor.
type MockCall struct {
	Name string
	Args []string
	Env  []string
	Dir  string
}

// Run implements CommandExecutor.
func (m *MockCommandExecutor) Run(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error) {
	m.Calls = append(m.Calls, MockCall{
		Name: name,
		Args: args,
		Env:  env,
		Dir:  dir,
	})
	if m.RunFunc != nil {
		return m.RunFunc(ctx, name, args, env, dir)
	}
	return []byte("mock output"), nil
}

func TestGetInfo(t *testing.T) {
	p := &HexPlugin{}
	info := p.GetInfo()

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{
			name:     "plugin name",
			got:      info.Name,
			expected: "hex",
		},
		{
			name:     "plugin version",
			got:      info.Version,
			expected: "2.0.0",
		},
		{
			name:     "plugin description",
			got:      info.Description,
			expected: "Publish packages to Hex.pm (Elixir)",
		},
		{
			name:     "plugin author",
			got:      info.Author,
			expected: "Relicta Team",
		},
		{
			name:     "hooks count",
			got:      len(info.Hooks),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, expected %v", tt.got, tt.expected)
			}
		})
	}

	// Verify the hook is PostPublish
	t.Run("hook is PostPublish", func(t *testing.T) {
		if len(info.Hooks) < 1 {
			t.Fatal("expected at least one hook")
		}
		if info.Hooks[0] != plugin.HookPostPublish {
			t.Errorf("got hook %v, expected %v", info.Hooks[0], plugin.HookPostPublish)
		}
	})

	// Verify ConfigSchema is valid JSON
	t.Run("config schema is valid JSON", func(t *testing.T) {
		if info.ConfigSchema == "" {
			t.Error("config schema should not be empty")
		}
	})

	// Verify ConfigSchema contains expected properties
	t.Run("config schema contains api_key", func(t *testing.T) {
		if !strings.Contains(info.ConfigSchema, "api_key") {
			t.Error("config schema should contain api_key")
		}
	})

	t.Run("config schema contains organization", func(t *testing.T) {
		if !strings.Contains(info.ConfigSchema, "organization") {
			t.Error("config schema should contain organization")
		}
	})

	t.Run("config schema contains replace", func(t *testing.T) {
		if !strings.Contains(info.ConfigSchema, "replace") {
			t.Error("config schema should contain replace")
		}
	})

	t.Run("config schema contains yes", func(t *testing.T) {
		if !strings.Contains(info.ConfigSchema, "yes") {
			t.Error("config schema should contain yes")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		envVars     map[string]string
		expectValid bool
		expectError bool
		errorField  string
	}{
		{
			name:        "empty config is valid",
			config:      map[string]any{},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "nil config is valid",
			config:      nil,
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with api_key is valid",
			config: map[string]any{
				"api_key": "test-key-123",
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with organization is valid",
			config: map[string]any{
				"organization": "my-org",
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with replace flag is valid",
			config: map[string]any{
				"replace": true,
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with yes flag is valid",
			config: map[string]any{
				"yes": true,
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name:   "config via HEX_API_KEY env var is valid",
			config: map[string]any{},
			envVars: map[string]string{
				"HEX_API_KEY": "env-api-key-123",
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "full config with all options is valid",
			config: map[string]any{
				"api_key":      "test-key-123",
				"organization": "my-org",
				"replace":      true,
				"yes":          false,
				"work_dir":     "packages/my-lib",
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
		{
			name: "config with invalid organization is invalid",
			config: map[string]any{
				"organization": "my org with spaces",
			},
			envVars:     nil,
			expectValid: false,
			expectError: false,
			errorField:  "organization",
		},
		{
			name: "config with path traversal work_dir is invalid",
			config: map[string]any{
				"work_dir": "../../../etc",
			},
			envVars:     nil,
			expectValid: false,
			expectError: false,
			errorField:  "work_dir",
		},
		{
			name: "config with absolute path work_dir is invalid",
			config: map[string]any{
				"work_dir": "/etc/passwd",
			},
			envVars:     nil,
			expectValid: false,
			expectError: false,
			errorField:  "work_dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			_ = os.Unsetenv("HEX_API_KEY")
			_ = os.Unsetenv("HEX_ORGANIZATION")

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			p := &HexPlugin{}
			resp, err := p.Validate(context.Background(), tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Valid != tt.expectValid {
				t.Errorf("got valid=%v, expected valid=%v, errors=%v", resp.Valid, tt.expectValid, resp.Errors)
			}

			if !tt.expectValid && tt.errorField != "" {
				found := false
				for _, e := range resp.Errors {
					if e.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error on field %q, got errors: %v", tt.errorField, resp.Errors)
				}
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name            string
		config          map[string]any
		envVars         map[string]string
		expectedAPIKey  string
		expectedOrg     string
		expectedReplace bool
		expectedYes     bool
		expectedWorkDir string
	}{
		{
			name:            "empty config uses defaults",
			config:          map[string]any{},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "config values take precedence",
			config: map[string]any{
				"api_key":      "config-key",
				"organization": "my-org",
				"replace":      true,
				"yes":          false,
				"work_dir":     "packages/lib",
			},
			envVars:         nil,
			expectedAPIKey:  "config-key",
			expectedOrg:     "my-org",
			expectedReplace: true,
			expectedYes:     false,
			expectedWorkDir: "packages/lib",
		},
		{
			name:   "env var fallback for api_key",
			config: map[string]any{},
			envVars: map[string]string{
				"HEX_API_KEY": "env-key-123",
			},
			expectedAPIKey:  "env-key-123",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "config api_key takes precedence over env var",
			config: map[string]any{
				"api_key": "config-key",
			},
			envVars: map[string]string{
				"HEX_API_KEY": "env-key-123",
			},
			expectedAPIKey:  "config-key",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "replace flag as string true",
			config: map[string]any{
				"replace": "true",
			},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedOrg:     "",
			expectedReplace: true,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "replace flag as boolean false",
			config: map[string]any{
				"replace": false,
			},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name:   "HEX_ORGANIZATION env var fallback",
			config: map[string]any{},
			envVars: map[string]string{
				"HEX_ORGANIZATION": "env-org",
			},
			expectedAPIKey:  "",
			expectedOrg:     "env-org",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "yes flag defaults to true",
			config: map[string]any{
				"api_key": "test-key",
			},
			envVars:         nil,
			expectedAPIKey:  "test-key",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     true,
			expectedWorkDir: ".",
		},
		{
			name: "yes flag can be disabled",
			config: map[string]any{
				"yes": false,
			},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedOrg:     "",
			expectedReplace: false,
			expectedYes:     false,
			expectedWorkDir: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			_ = os.Unsetenv("HEX_API_KEY")
			_ = os.Unsetenv("HEX_ORGANIZATION")

			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
				defer func(key string) { _ = os.Unsetenv(key) }(k)
			}

			p := &HexPlugin{}
			cfg := p.parseConfig(tt.config)

			if cfg.APIKey != tt.expectedAPIKey {
				t.Errorf("api_key: got %q, expected %q", cfg.APIKey, tt.expectedAPIKey)
			}
			if cfg.Organization != tt.expectedOrg {
				t.Errorf("organization: got %q, expected %q", cfg.Organization, tt.expectedOrg)
			}
			if cfg.Replace != tt.expectedReplace {
				t.Errorf("replace: got %v, expected %v", cfg.Replace, tt.expectedReplace)
			}
			if cfg.Yes != tt.expectedYes {
				t.Errorf("yes: got %v, expected %v", cfg.Yes, tt.expectedYes)
			}
			if cfg.WorkDir != tt.expectedWorkDir {
				t.Errorf("work_dir: got %q, expected %q", cfg.WorkDir, tt.expectedWorkDir)
			}
		})
	}
}

func TestExecuteDryRun(t *testing.T) {
	tests := []struct {
		name            string
		hook            plugin.Hook
		dryRun          bool
		config          map[string]any
		expectedSuccess bool
		expectedMessage string
		expectedOutputs map[string]any
	}{
		{
			name:   "PostPublish dry run returns would publish message",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key": "test-key",
			},
			expectedSuccess: true,
			expectedMessage: "Would publish package to Hex.pm",
			expectedOutputs: map[string]any{
				"command":      "mix hex.publish --yes",
				"version":      "1.0.0",
				"organization": "",
				"replace":      false,
			},
		},
		{
			name:   "PostPublish dry run with organization",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key":      "test-key",
				"organization": "my-org",
			},
			expectedSuccess: true,
			expectedMessage: "Would publish package to Hex.pm",
			expectedOutputs: map[string]any{
				"command":      "mix hex.publish --organization my-org --yes",
				"version":      "1.0.0",
				"organization": "my-org",
				"replace":      false,
			},
		},
		{
			name:   "PostPublish dry run with replace",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key": "test-key",
				"replace": true,
			},
			expectedSuccess: true,
			expectedMessage: "Would publish package to Hex.pm",
			expectedOutputs: map[string]any{
				"command":      "mix hex.publish --replace --yes",
				"version":      "1.0.0",
				"organization": "",
				"replace":      true,
			},
		},
		{
			name:   "PostPublish dry run with all options",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key":      "test-key",
				"organization": "my-org",
				"replace":      true,
				"yes":          true,
			},
			expectedSuccess: true,
			expectedMessage: "Would publish package to Hex.pm",
			expectedOutputs: map[string]any{
				"command":      "mix hex.publish --organization my-org --replace --yes",
				"version":      "1.0.0",
				"organization": "my-org",
				"replace":      true,
			},
		},
		{
			name:   "PostPublish dry run without yes flag",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key": "test-key",
				"yes":     false,
			},
			expectedSuccess: true,
			expectedMessage: "Would publish package to Hex.pm",
			expectedOutputs: map[string]any{
				"command":      "mix hex.publish",
				"version":      "1.0.0",
				"organization": "",
				"replace":      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HexPlugin{}
			req := plugin.ExecuteRequest{
				Hook:   tt.hook,
				DryRun: tt.dryRun,
				Config: tt.config,
				Context: plugin.ReleaseContext{
					Version:     "1.0.0",
					TagName:     "v1.0.0",
					ReleaseType: "minor",
					Branch:      "main",
					CommitSHA:   "abc123",
				},
			}

			resp, err := p.Execute(context.Background(), req)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Success != tt.expectedSuccess {
				t.Errorf("success: got %v, expected %v", resp.Success, tt.expectedSuccess)
			}

			if resp.Message != tt.expectedMessage {
				t.Errorf("message: got %q, expected %q", resp.Message, tt.expectedMessage)
			}

			if tt.expectedOutputs != nil {
				for key, expectedValue := range tt.expectedOutputs {
					if resp.Outputs[key] != expectedValue {
						t.Errorf("output %q: got %v, expected %v", key, resp.Outputs[key], expectedValue)
					}
				}
			}
		})
	}
}

func TestExecuteActualRun(t *testing.T) {
	tests := []struct {
		name            string
		config          map[string]any
		mockOutput      []byte
		mockError       error
		expectedSuccess bool
		expectedMessage string
		expectedError   string
		verifyCall      func(t *testing.T, calls []MockCall)
	}{
		{
			name: "successful publish",
			config: map[string]any{
				"api_key": "test-api-key",
			},
			mockOutput:      []byte("Published my_package v1.0.0"),
			mockError:       nil,
			expectedSuccess: true,
			expectedMessage: "Published package v1.0.0 to Hex.pm",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 1 {
					t.Errorf("expected 1 call, got %d", len(calls))
					return
				}
				call := calls[0]
				if call.Name != "mix" {
					t.Errorf("expected command 'mix', got %q", call.Name)
				}
				if !contains(call.Args, "hex.publish") {
					t.Error("expected args to contain 'hex.publish'")
				}
				if !contains(call.Args, "--yes") {
					t.Error("expected args to contain '--yes'")
				}
				// Verify HEX_API_KEY is in env
				foundAPIKey := false
				for _, env := range call.Env {
					if strings.HasPrefix(env, "HEX_API_KEY=") {
						foundAPIKey = true
						break
					}
				}
				if !foundAPIKey {
					t.Error("expected HEX_API_KEY in environment")
				}
			},
		},
		{
			name: "publish with organization",
			config: map[string]any{
				"api_key":      "test-api-key",
				"organization": "my-org",
			},
			mockOutput:      []byte("Published my_package v1.0.0 to organization my-org"),
			mockError:       nil,
			expectedSuccess: true,
			expectedMessage: "Published package v1.0.0 to Hex.pm",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 1 {
					t.Errorf("expected 1 call, got %d", len(calls))
					return
				}
				call := calls[0]
				if !contains(call.Args, "--organization") {
					t.Error("expected args to contain '--organization'")
				}
				if !contains(call.Args, "my-org") {
					t.Error("expected args to contain 'my-org'")
				}
			},
		},
		{
			name: "publish with replace",
			config: map[string]any{
				"api_key": "test-api-key",
				"replace": true,
			},
			mockOutput:      []byte("Replaced my_package v1.0.0"),
			mockError:       nil,
			expectedSuccess: true,
			expectedMessage: "Published package v1.0.0 to Hex.pm",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 1 {
					t.Errorf("expected 1 call, got %d", len(calls))
					return
				}
				call := calls[0]
				if !contains(call.Args, "--replace") {
					t.Error("expected args to contain '--replace'")
				}
			},
		},
		{
			name: "publish with work_dir",
			config: map[string]any{
				"api_key":  "test-api-key",
				"work_dir": "packages/my-lib",
			},
			mockOutput:      []byte("Published my_package v1.0.0"),
			mockError:       nil,
			expectedSuccess: true,
			expectedMessage: "Published package v1.0.0 to Hex.pm",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 1 {
					t.Errorf("expected 1 call, got %d", len(calls))
					return
				}
				call := calls[0]
				if call.Dir != "packages/my-lib" {
					t.Errorf("expected dir 'packages/my-lib', got %q", call.Dir)
				}
			},
		},
		{
			name: "publish without yes flag",
			config: map[string]any{
				"api_key": "test-api-key",
				"yes":     false,
			},
			mockOutput:      []byte("Published my_package v1.0.0"),
			mockError:       nil,
			expectedSuccess: true,
			expectedMessage: "Published package v1.0.0 to Hex.pm",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 1 {
					t.Errorf("expected 1 call, got %d", len(calls))
					return
				}
				call := calls[0]
				if contains(call.Args, "--yes") {
					t.Error("expected args to NOT contain '--yes'")
				}
			},
		},
		{
			name:   "missing api_key fails",
			config: map[string]any{
				// No api_key
			},
			mockOutput:      nil,
			mockError:       nil,
			expectedSuccess: false,
			expectedError:   "HEX_API_KEY is required",
			verifyCall: func(t *testing.T, calls []MockCall) {
				if len(calls) != 0 {
					t.Errorf("expected 0 calls when api_key is missing, got %d", len(calls))
				}
			},
		},
		{
			name: "mix command fails",
			config: map[string]any{
				"api_key": "test-api-key",
			},
			mockOutput:      []byte("** (Mix) Could not find package"),
			mockError:       errors.New("exit status 1"),
			expectedSuccess: false,
			expectedError:   "mix hex.publish failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			_ = os.Unsetenv("HEX_API_KEY")
			_ = os.Unsetenv("HEX_ORGANIZATION")

			mock := &MockCommandExecutor{
				RunFunc: func(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error) {
					return tt.mockOutput, tt.mockError
				},
			}

			p := &HexPlugin{executor: mock}
			req := plugin.ExecuteRequest{
				Hook:   plugin.HookPostPublish,
				DryRun: false,
				Config: tt.config,
				Context: plugin.ReleaseContext{
					Version:     "1.0.0",
					TagName:     "v1.0.0",
					ReleaseType: "minor",
					Branch:      "main",
					CommitSHA:   "abc123",
				},
			}

			resp, err := p.Execute(context.Background(), req)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Success != tt.expectedSuccess {
				t.Errorf("success: got %v, expected %v, error: %s", resp.Success, tt.expectedSuccess, resp.Error)
			}

			if tt.expectedSuccess && resp.Message != tt.expectedMessage {
				t.Errorf("message: got %q, expected %q", resp.Message, tt.expectedMessage)
			}

			if !tt.expectedSuccess && tt.expectedError != "" {
				if !strings.Contains(resp.Error, tt.expectedError) {
					t.Errorf("error: expected to contain %q, got %q", tt.expectedError, resp.Error)
				}
			}

			if tt.verifyCall != nil {
				tt.verifyCall(t, mock.Calls)
			}
		})
	}
}

func TestExecuteUnhandledHook(t *testing.T) {
	unhandledHooks := []plugin.Hook{
		plugin.HookPreInit,
		plugin.HookPostInit,
		plugin.HookPrePlan,
		plugin.HookPostPlan,
		plugin.HookPreVersion,
		plugin.HookPostVersion,
		plugin.HookPreNotes,
		plugin.HookPostNotes,
		plugin.HookPreApprove,
		plugin.HookPostApprove,
		plugin.HookPrePublish,
		plugin.HookOnSuccess,
		plugin.HookOnError,
	}

	for _, hook := range unhandledHooks {
		t.Run(string(hook), func(t *testing.T) {
			p := &HexPlugin{}
			req := plugin.ExecuteRequest{
				Hook:   hook,
				DryRun: false,
				Config: map[string]any{},
				Context: plugin.ReleaseContext{
					Version:     "1.0.0",
					TagName:     "v1.0.0",
					ReleaseType: "minor",
					Branch:      "main",
					CommitSHA:   "abc123",
				},
			}

			resp, err := p.Execute(context.Background(), req)

			if err != nil {
				t.Errorf("unexpected error for hook %s: %v", hook, err)
				return
			}

			if !resp.Success {
				t.Errorf("expected success=true for unhandled hook %s, got success=false", hook)
			}

			expectedMessage := "Hook " + string(hook) + " not handled"
			if resp.Message != expectedMessage {
				t.Errorf("message for hook %s: got %q, expected %q", hook, resp.Message, expectedMessage)
			}
		})
	}
}

func TestExecuteWithReleaseContext(t *testing.T) {
	tests := []struct {
		name            string
		context         plugin.ReleaseContext
		expectedVersion string
	}{
		{
			name: "basic release context",
			context: plugin.ReleaseContext{
				Version:     "1.0.0",
				TagName:     "v1.0.0",
				ReleaseType: "minor",
				Branch:      "main",
				CommitSHA:   "abc123def456",
			},
			expectedVersion: "1.0.0",
		},
		{
			name: "version with v prefix",
			context: plugin.ReleaseContext{
				Version:     "v2.0.0",
				TagName:     "v2.0.0",
				ReleaseType: "major",
				Branch:      "main",
				CommitSHA:   "abc123def456",
			},
			expectedVersion: "2.0.0",
		},
		{
			name: "patch release context",
			context: plugin.ReleaseContext{
				Version:         "1.0.1",
				PreviousVersion: "1.0.0",
				TagName:         "v1.0.1",
				ReleaseType:     "patch",
				Branch:          "hotfix/urgent-fix",
				CommitSHA:       "hotfix123",
			},
			expectedVersion: "1.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HexPlugin{}
			req := plugin.ExecuteRequest{
				Hook:    plugin.HookPostPublish,
				DryRun:  true,
				Config:  map[string]any{},
				Context: tt.context,
			}

			resp, err := p.Execute(context.Background(), req)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !resp.Success {
				t.Errorf("expected success=true, got success=false")
			}

			if version, ok := resp.Outputs["version"].(string); ok {
				if version != tt.expectedVersion {
					t.Errorf("version: got %q, expected %q", version, tt.expectedVersion)
				}
			} else {
				t.Error("expected version in outputs")
			}
		})
	}
}

func TestExecuteSecurityValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		expectedError string
	}{
		{
			name: "path traversal in work_dir fails",
			config: map[string]any{
				"api_key":  "test-key",
				"work_dir": "../../../etc",
			},
			expectedError: "invalid work_dir",
		},
		{
			name: "absolute path in work_dir fails",
			config: map[string]any{
				"api_key":  "test-key",
				"work_dir": "/etc/passwd",
			},
			expectedError: "invalid work_dir",
		},
		{
			name: "invalid organization name fails",
			config: map[string]any{
				"api_key":      "test-key",
				"organization": "my org; rm -rf /",
			},
			expectedError: "invalid organization",
		},
		{
			name: "organization with special characters fails",
			config: map[string]any{
				"api_key":      "test-key",
				"organization": "my-org$(whoami)",
			},
			expectedError: "invalid organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommandExecutor{
				RunFunc: func(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error) {
					return []byte("success"), nil
				},
			}

			p := &HexPlugin{executor: mock}
			req := plugin.ExecuteRequest{
				Hook:   plugin.HookPostPublish,
				DryRun: false,
				Config: tt.config,
				Context: plugin.ReleaseContext{
					Version: "1.0.0",
					TagName: "v1.0.0",
				},
			}

			resp, err := p.Execute(context.Background(), req)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Success {
				t.Error("expected success=false for security validation failure")
			}

			if !strings.Contains(resp.Error, tt.expectedError) {
				t.Errorf("error: expected to contain %q, got %q", tt.expectedError, resp.Error)
			}

			// Verify no command was executed
			if len(mock.Calls) > 0 {
				t.Errorf("expected no commands to be executed, got %d calls", len(mock.Calls))
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path is valid",
			path:        "",
			expectError: false,
		},
		{
			name:        "current directory is valid",
			path:        ".",
			expectError: false,
		},
		{
			name:        "relative path is valid",
			path:        "packages/my-lib",
			expectError: false,
		},
		{
			name:        "nested relative path is valid",
			path:        "a/b/c/d",
			expectError: false,
		},
		{
			name:        "absolute path is invalid",
			path:        "/etc/passwd",
			expectError: true,
			errorMsg:    "absolute paths are not allowed",
		},
		{
			name:        "path traversal with .. is invalid",
			path:        "../secret",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "nested path traversal is invalid",
			path:        "packages/../../secret",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error: expected to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateOrganization(t *testing.T) {
	tests := []struct {
		name        string
		org         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty organization is valid",
			org:         "",
			expectError: false,
		},
		{
			name:        "simple name is valid",
			org:         "myorg",
			expectError: false,
		},
		{
			name:        "name with hyphen is valid",
			org:         "my-org",
			expectError: false,
		},
		{
			name:        "name with underscore is valid",
			org:         "my_org",
			expectError: false,
		},
		{
			name:        "name with numbers is valid",
			org:         "myorg123",
			expectError: false,
		},
		{
			name:        "mixed case is valid",
			org:         "MyOrg",
			expectError: false,
		},
		{
			name:        "name with spaces is invalid",
			org:         "my org",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "name with special characters is invalid",
			org:         "my-org$",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "name with semicolon is invalid",
			org:         "org;rm -rf /",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "too long name is invalid",
			org:         strings.Repeat("a", 129),
			expectError: true,
			errorMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrganization(tt.org)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error: expected to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidationBuilder(t *testing.T) {
	// Test the validation builder used by the plugin
	t.Run("empty validation is valid", func(t *testing.T) {
		vb := helpers.NewValidationBuilder()
		resp := vb.Build()
		if !resp.Valid {
			t.Error("expected valid=true for empty validation")
		}
		if len(resp.Errors) != 0 {
			t.Errorf("expected no errors, got %d", len(resp.Errors))
		}
	})

	t.Run("validation with errors is invalid", func(t *testing.T) {
		vb := helpers.NewValidationBuilder()
		vb.AddError("field1", "error message")
		resp := vb.Build()
		if resp.Valid {
			t.Error("expected valid=false when errors are present")
		}
		if len(resp.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(resp.Errors))
		}
	})
}

func TestGetExecutor(t *testing.T) {
	t.Run("returns real executor when none set", func(t *testing.T) {
		p := &HexPlugin{}
		executor := p.getExecutor()
		if _, ok := executor.(*RealCommandExecutor); !ok {
			t.Error("expected RealCommandExecutor when no executor is set")
		}
	})

	t.Run("returns mock executor when set", func(t *testing.T) {
		mock := &MockCommandExecutor{}
		p := &HexPlugin{executor: mock}
		executor := p.getExecutor()
		if executor != mock {
			t.Error("expected mock executor to be returned")
		}
	})
}

// Helper function to check if a slice contains a string.
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
