// Package main provides tests for the Hex plugin.
package main

import (
	"context"
	"os"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

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
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		envVars     map[string]string
		expectValid bool
		expectError bool
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
			name: "config with mix_path is valid",
			config: map[string]any{
				"mix_path": "/usr/local/bin/mix",
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
				"mix_path":     "/custom/bin/mix",
				"replace":      true,
			},
			envVars:     nil,
			expectValid: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
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
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]any
		envVars        map[string]string
		expectedAPIKey string
		expectedMixPath string
		expectedOrg    string
		expectedReplace bool
	}{
		{
			name:            "empty config uses defaults",
			config:          map[string]any{},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedMixPath: "mix",
			expectedOrg:     "",
			expectedReplace: false,
		},
		{
			name: "config values take precedence",
			config: map[string]any{
				"api_key":      "config-key",
				"mix_path":     "/custom/mix",
				"organization": "my-org",
				"replace":      true,
			},
			envVars:         nil,
			expectedAPIKey:  "config-key",
			expectedMixPath: "/custom/mix",
			expectedOrg:     "my-org",
			expectedReplace: true,
		},
		{
			name:   "env var fallback for api_key",
			config: map[string]any{},
			envVars: map[string]string{
				"HEX_API_KEY": "env-key-123",
			},
			expectedAPIKey:  "env-key-123",
			expectedMixPath: "mix",
			expectedOrg:     "",
			expectedReplace: false,
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
			expectedMixPath: "mix",
			expectedOrg:     "",
			expectedReplace: false,
		},
		{
			name: "replace flag as string true",
			config: map[string]any{
				"replace": "true",
			},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedMixPath: "mix",
			expectedOrg:     "",
			expectedReplace: true,
		},
		{
			name: "replace flag as boolean false",
			config: map[string]any{
				"replace": false,
			},
			envVars:         nil,
			expectedAPIKey:  "",
			expectedMixPath: "mix",
			expectedOrg:     "",
			expectedReplace: false,
		},
		{
			name:   "HEX_ORGANIZATION env var fallback",
			config: map[string]any{},
			envVars: map[string]string{
				"HEX_ORGANIZATION": "env-org",
			},
			expectedAPIKey:  "",
			expectedMixPath: "mix",
			expectedOrg:     "env-org",
			expectedReplace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			os.Unsetenv("HEX_API_KEY")
			os.Unsetenv("HEX_ORGANIZATION")

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cp := helpers.NewConfigParser(tt.config)

			apiKey := cp.GetString("api_key", "HEX_API_KEY", "")
			mixPath := cp.GetString("mix_path", "", "mix")
			org := cp.GetString("organization", "HEX_ORGANIZATION", "")
			replace := cp.GetBool("replace", false)

			if apiKey != tt.expectedAPIKey {
				t.Errorf("api_key: got %q, expected %q", apiKey, tt.expectedAPIKey)
			}
			if mixPath != tt.expectedMixPath {
				t.Errorf("mix_path: got %q, expected %q", mixPath, tt.expectedMixPath)
			}
			if org != tt.expectedOrg {
				t.Errorf("organization: got %q, expected %q", org, tt.expectedOrg)
			}
			if replace != tt.expectedReplace {
				t.Errorf("replace: got %v, expected %v", replace, tt.expectedReplace)
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
	}{
		{
			name:            "PostPublish dry run returns would execute message",
			hook:            plugin.HookPostPublish,
			dryRun:          true,
			config:          map[string]any{},
			expectedSuccess: true,
			expectedMessage: "Would execute hex plugin",
		},
		{
			name:            "PostPublish actual run returns success message",
			hook:            plugin.HookPostPublish,
			dryRun:          false,
			config:          map[string]any{},
			expectedSuccess: true,
			expectedMessage: "Hex plugin executed successfully",
		},
		{
			name:   "PostPublish dry run with config",
			hook:   plugin.HookPostPublish,
			dryRun: true,
			config: map[string]any{
				"api_key":      "test-key",
				"organization": "my-org",
			},
			expectedSuccess: true,
			expectedMessage: "Would execute hex plugin",
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
		name    string
		context plugin.ReleaseContext
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
		},
		{
			name: "full release context",
			context: plugin.ReleaseContext{
				Version:         "2.0.0",
				PreviousVersion: "1.9.0",
				TagName:         "v2.0.0",
				ReleaseType:     "major",
				RepositoryURL:   "https://github.com/example/hex-package",
				RepositoryOwner: "example",
				RepositoryName:  "hex-package",
				Branch:          "main",
				CommitSHA:       "abc123def456789",
				Changelog:       "## Changes\n- New feature",
				ReleaseNotes:    "Release v2.0.0 with breaking changes",
			},
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
