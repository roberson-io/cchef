package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/roberson-io/cchef/core"
)

// cchef reads a config file only for settings that belong to the machine rather
// than to a recipe. There is one so far: which CyberChef instance share links
// point at, which a self-hosted or air-gapped deployment has to change. Recipes
// and their arguments never come from here — a recipe has to mean the same
// thing wherever it is run.

const (
	// configEnvVar names the config file outright, bypassing the search.
	configEnvVar = "CCHEF_CONFIG"
	// baseURLEnvVar sets the CyberChef instance for one invocation.
	baseURLEnvVar = "CCHEF_BASE_URL"
	// configDirName is cchef's directory under the config home.
	configDirName = "cchef"
	// configFileName is the file within it.
	configFileName = "config.yaml"
)

// config is the on-disk configuration. Keys match their flag names so a setting
// is looked up the same way whether it is typed or stored.
type config struct {
	BaseURL string `yaml:"base-url"`
}

// configPath returns where the config file is read from: CCHEF_CONFIG if set,
// otherwise `cchef/config.yaml` under $XDG_CONFIG_HOME, which defaults to
// ~/.config as the XDG Base Directory Specification says. The XDG layout is
// used on every platform so the path is the same one everywhere.
func configPath() string {
	if p := os.Getenv(configEnvVar); p != "" {
		return p
	}
	home := os.Getenv("XDG_CONFIG_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			// With no home directory there is nowhere to look; the caller
			// treats an unreadable path as no config at all.
			return ""
		}
		home = filepath.Join(userHome, ".config")
	}
	return filepath.Join(home, configDirName, configFileName)
}

// loadConfig reads the config file. Having none is the normal case and returns
// the zero config; a file that exists but cannot be read or parsed is an error
// naming it, since silently ignoring it would hide a typo.
func loadConfig() (config, error) {
	var cfg config
	path := configPath()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- reads cchef's own config file, at a path the user controls by design
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// flagBaseURL holds --base-url.
var flagBaseURL string

// addBaseURLFlag registers --base-url on a command that generates share links.
func addBaseURLFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flagBaseURL, "base-url", "",
		"CyberChef instance to link to (default "+core.DefaultBaseURL+")")
}

// resolveBaseURL settles which CyberChef instance share links point at, in
// order of how specific the source is: the flag, then the environment, then the
// config file, then the public instance. Whichever supplies it is named in the
// error if it is unusable, since otherwise the link would simply go nowhere.
func resolveBaseURL(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("base-url") {
		return checkBaseURL(flagBaseURL, "--base-url")
	}
	if env := os.Getenv(baseURLEnvVar); env != "" {
		return checkBaseURL(env, baseURLEnvVar)
	}
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.BaseURL != "" {
		return checkBaseURL(cfg.BaseURL, configPath())
	}
	return core.DefaultBaseURL, nil
}

// checkBaseURL accepts an absolute http or https address, naming source in the
// error so the value can be found and fixed.
func checkBaseURL(value, source string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%s: %q is not a URL: %w", source, value, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%s: %q needs an http:// or https:// scheme", source, value)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%s: %q names no host", source, value)
	}
	return value, nil
}
