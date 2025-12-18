// Package main implements the Hex plugin for Relicta.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	Run(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error)
}

// RealCommandExecutor executes actual system commands.
type RealCommandExecutor struct{}

// Run executes the command with the given arguments.
func (e *RealCommandExecutor) Run(ctx context.Context, name string, args []string, env []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.CombinedOutput()
}

// Config represents the Hex plugin configuration.
type Config struct {
	APIKey       string
	Organization string
	Replace      bool
	Yes          bool
	WorkDir      string
}

// HexPlugin implements the Publish packages to Hex.pm (Elixir) plugin.
type HexPlugin struct {
	executor CommandExecutor
}

// getExecutor returns the command executor, defaulting to RealCommandExecutor.
func (p *HexPlugin) getExecutor() CommandExecutor {
	if p.executor != nil {
		return p.executor
	}
	return &RealCommandExecutor{}
}

// GetInfo returns plugin metadata.
func (p *HexPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "hex",
		Version:     "2.0.0",
		Description: "Publish packages to Hex.pm (Elixir)",
		Author:      "Relicta Team",
		Hooks: []plugin.Hook{
			plugin.HookPostPublish,
		},
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"api_key": {"type": "string", "description": "Hex.pm API key (or use HEX_API_KEY env)"},
				"organization": {"type": "string", "description": "Hex.pm organization for private packages"},
				"replace": {"type": "boolean", "description": "Replace existing package version", "default": false},
				"yes": {"type": "boolean", "description": "Skip confirmation prompt", "default": true},
				"work_dir": {"type": "string", "description": "Working directory for mix command", "default": "."}
			}
		}`,
	}
}

// validatePath validates a file path to prevent path traversal.
func validatePath(path string) error {
	if path == "" {
		return nil
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Check for absolute paths (potential escape from working directory)
	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("absolute paths are not allowed")
	}

	// Check for path traversal attempts
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return fmt.Errorf("path traversal detected: cannot use '..' to escape working directory")
	}

	return nil
}

// validateOrganization validates organization name format.
func validateOrganization(org string) error {
	if org == "" {
		return nil
	}

	if len(org) > 128 {
		return fmt.Errorf("organization name too long (max 128 characters)")
	}

	// Organization names should be alphanumeric with hyphens and underscores
	for _, r := range org {
		isLower := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		isHyphen := r == '-'
		isUnderscore := r == '_'

		if !isLower && !isUpper && !isDigit && !isHyphen && !isUnderscore {
			return fmt.Errorf("organization name contains invalid characters: only alphanumeric, hyphens, and underscores are allowed")
		}
	}

	return nil
}

// parseConfig parses the raw configuration into a typed Config struct.
func (p *HexPlugin) parseConfig(raw map[string]any) *Config {
	parser := helpers.NewConfigParser(raw)

	return &Config{
		APIKey:       parser.GetString("api_key", "HEX_API_KEY", ""),
		Organization: parser.GetString("organization", "HEX_ORGANIZATION", ""),
		Replace:      parser.GetBool("replace", false),
		Yes:          parser.GetBool("yes", true),
		WorkDir:      parser.GetString("work_dir", "", "."),
	}
}

// Execute runs the plugin for a given hook.
func (p *HexPlugin) Execute(ctx context.Context, req plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	cfg := p.parseConfig(req.Config)

	switch req.Hook {
	case plugin.HookPostPublish:
		return p.publish(ctx, cfg, req.Context, req.DryRun)
	default:
		return &plugin.ExecuteResponse{
			Success: true,
			Message: fmt.Sprintf("Hook %s not handled", req.Hook),
		}, nil
	}
}

// publish executes mix hex.publish to publish the package to Hex.pm.
func (p *HexPlugin) publish(ctx context.Context, cfg *Config, releaseCtx plugin.ReleaseContext, dryRun bool) (*plugin.ExecuteResponse, error) {
	// Validate configuration
	if err := validatePath(cfg.WorkDir); err != nil {
		return &plugin.ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid work_dir: %v", err),
		}, nil
	}

	if err := validateOrganization(cfg.Organization); err != nil {
		return &plugin.ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid organization: %v", err),
		}, nil
	}

	// Build command arguments
	args := []string{"hex.publish"}

	if cfg.Organization != "" {
		args = append(args, "--organization", cfg.Organization)
	}

	if cfg.Replace {
		args = append(args, "--replace")
	}

	if cfg.Yes {
		args = append(args, "--yes")
	}

	version := strings.TrimPrefix(releaseCtx.Version, "v")

	if dryRun {
		return &plugin.ExecuteResponse{
			Success: true,
			Message: "Would publish package to Hex.pm",
			Outputs: map[string]any{
				"command":      "mix " + strings.Join(args, " "),
				"version":      version,
				"organization": cfg.Organization,
				"replace":      cfg.Replace,
			},
		}, nil
	}

	// Check for API key
	if cfg.APIKey == "" {
		return &plugin.ExecuteResponse{
			Success: false,
			Error:   "HEX_API_KEY is required: set api_key in config or HEX_API_KEY environment variable",
		}, nil
	}

	// Build environment with HEX_API_KEY
	env := []string{
		fmt.Sprintf("HEX_API_KEY=%s", cfg.APIKey),
	}

	// Execute mix hex.publish
	output, err := p.getExecutor().Run(ctx, "mix", args, env, cfg.WorkDir)
	if err != nil {
		return &plugin.ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("mix hex.publish failed: %v\nOutput: %s", err, string(output)),
		}, nil
	}

	return &plugin.ExecuteResponse{
		Success: true,
		Message: fmt.Sprintf("Published package v%s to Hex.pm", version),
		Outputs: map[string]any{
			"version":      version,
			"organization": cfg.Organization,
			"output":       string(output),
		},
	}, nil
}

// Validate validates the plugin configuration.
func (p *HexPlugin) Validate(_ context.Context, config map[string]any) (*plugin.ValidateResponse, error) {
	vb := helpers.NewValidationBuilder()
	parser := helpers.NewConfigParser(config)

	// Validate work_dir if provided
	workDir := parser.GetString("work_dir", "", ".")
	if err := validatePath(workDir); err != nil {
		vb.AddError("work_dir", err.Error())
	}

	// Validate organization if provided
	org := parser.GetString("organization", "HEX_ORGANIZATION", "")
	if err := validateOrganization(org); err != nil {
		vb.AddError("organization", err.Error())
	}

	return vb.Build(), nil
}
