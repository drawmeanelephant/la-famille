package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

type cliBootstrap struct {
	ProjectRoot         string
	ConfigPath          string
	explicitProjectRoot bool
}

// bootstrapCLIArgs extracts the two flags needed before Cobra can construct
// command defaults. It intentionally accepts both --flag value and
// --flag=value, matching pflag's user-facing forms.
func bootstrapCLIArgs(args []string) (cliBootstrap, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return cliBootstrap{}, fmt.Errorf("get current directory: %w", err)
	}

	var projectRootArg, configArg string
	var projectRootSet, configSet bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		for _, name := range []string{"--project-root", "--config"} {
			if arg == name {
				if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
					return cliBootstrap{}, fmt.Errorf("%s requires a path", name)
				}
				value := args[i+1]
				i++
				if name == "--project-root" {
					projectRootArg, projectRootSet = value, true
				} else {
					configArg, configSet = value, true
				}
				continue
			}
			prefix := name + "="
			if strings.HasPrefix(arg, prefix) {
				value := strings.TrimPrefix(arg, prefix)
				if strings.TrimSpace(value) == "" {
					return cliBootstrap{}, fmt.Errorf("%s requires a path", name)
				}
				if name == "--project-root" {
					projectRootArg, projectRootSet = value, true
				} else {
					configArg, configSet = value, true
				}
			}
		}
	}

	root := cwd
	if projectRootSet {
		root, err = absolutePath(cwd, projectRootArg)
		if err != nil {
			return cliBootstrap{}, fmt.Errorf("resolve --project-root: %w", err)
		}
	}

	configPath := ""
	if configSet {
		configPath, err = absolutePath(cwd, configArg)
		if err != nil {
			return cliBootstrap{}, fmt.Errorf("resolve --config: %w", err)
		}
	} else {
		configPath = filepath.Join(root, "config.yaml")
	}

	return cliBootstrap{ProjectRoot: root, ConfigPath: configPath, explicitProjectRoot: projectRootSet}, nil
}

// loadProjectConfig implements the runtime precedence contract:
// explicit --project-root > config.yaml project_root > config file directory
// (or CWD when no config file is selected), and all configured relative paths
// are then resolved from that root.
func loadProjectConfig(args []string) (config.Config, error) {
	boot, err := bootstrapCLIArgs(args)
	if err != nil {
		return config.Config{}, err
	}

	raw, loadErr := config.Load(boot.ConfigPath)
	root := boot.ProjectRoot
	if !boot.explicitProjectRoot && loadErr == nil {
		configuredRoot := strings.TrimSpace(raw.ProjectRoot)
		if configuredRoot != "" && configuredRoot != "." {
			root, err = absolutePath(filepath.Dir(boot.ConfigPath), configuredRoot)
			if err != nil {
				return config.Config{ProjectRoot: root, ConfigPath: boot.ConfigPath}, fmt.Errorf("resolve project_root: %w", err)
			}
		} else {
			root = filepath.Dir(boot.ConfigPath)
		}
	}

	base := config.Config{ProjectRoot: root, ConfigPath: boot.ConfigPath}
	if loadErr != nil {
		return base, fmt.Errorf("failed to load %s: %w", boot.ConfigPath, loadErr)
	}
	if err := raw.Validate(); err != nil {
		return base, fmt.Errorf("invalid %s: %w", boot.ConfigPath, err)
	}

	resolved, err := raw.ResolvePaths(root)
	if err != nil {
		return base, err
	}
	resolved.ConfigPath = boot.ConfigPath
	if err := resolved.ValidateResolved(); err != nil {
		return resolved, fmt.Errorf("invalid %s: %w", boot.ConfigPath, err)
	}
	return resolved, nil
}

func absolutePath(base, value string) (string, error) {
	if filepath.IsAbs(value) {
		return filepath.Abs(filepath.Clean(value))
	}
	return filepath.Abs(filepath.Join(base, value))
}

func resolveProjectPath(projectRoot, value string) string {
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	if projectRoot == "" || projectRoot == "." {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(projectRoot, value))
}
