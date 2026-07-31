package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

// writeConfig puts a config file in a temporary directory and points
// CCHEF_CONFIG at it.
func writeConfig(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(configEnvVar, path)
}

// newURLCmd builds a command carrying the base-url flag, as `cchef url` does.
func newURLCmd(t *testing.T) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "url"}
	addBaseURLFlag(c)
	return c
}

func TestConfigPathPrecedence(t *testing.T) {
	t.Run("CCHEF_CONFIG wins", func(t *testing.T) {
		t.Setenv(configEnvVar, "/tmp/somewhere/else.yaml")
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
		if got := configPath(); got != "/tmp/somewhere/else.yaml" {
			t.Errorf("configPath() = %q", got)
		}
	})
	t.Run("XDG_CONFIG_HOME next", func(t *testing.T) {
		t.Setenv(configEnvVar, "")
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
		if want := "/tmp/xdg/cchef/config.yaml"; configPath() != want {
			t.Errorf("configPath() = %q, want %q", configPath(), want)
		}
	})
	t.Run("~/.config by default", func(t *testing.T) {
		t.Setenv(configEnvVar, "")
		t.Setenv("XDG_CONFIG_HOME", "")
		got := configPath()
		if !strings.HasSuffix(got, filepath.Join(".config", "cchef", "config.yaml")) {
			t.Errorf("configPath() = %q, want it under ~/.config/cchef", got)
		}
	})
}

// TestLoadConfigMissingFile checks that having no config file is normal rather
// than an error — the tool has to work before anyone writes one.
func TestLoadConfigMissingFile(t *testing.T) {
	t.Setenv(configEnvVar, filepath.Join(t.TempDir(), "absent.yaml"))
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("a missing config file should not be an error: %v", err)
	}
	if cfg.BaseURL != "" {
		t.Errorf("BaseURL = %q, want empty", cfg.BaseURL)
	}
}

// TestLoadConfigMalformed checks that a broken config file is reported with its
// path, rather than silently ignored.
func TestLoadConfigMalformed(t *testing.T) {
	writeConfig(t, "base-url: [this is not a string\n")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("a malformed config file should be an error")
	}
	if !strings.Contains(err.Error(), "config.yaml") {
		t.Errorf("the error should name the file: %v", err)
	}
}

// TestResolveBaseURLPrecedence pins flag > environment > config file > default.
func TestResolveBaseURLPrecedence(t *testing.T) {
	const (
		fromFlag   = "https://flag.example/"
		fromEnv    = "https://env.example/"
		fromConfig = "https://config.example/"
	)

	t.Run("default when nothing is set", func(t *testing.T) {
		t.Setenv(configEnvVar, filepath.Join(t.TempDir(), "absent.yaml"))
		t.Setenv(baseURLEnvVar, "")
		got, err := resolveBaseURL(newURLCmd(t))
		if err != nil {
			t.Fatal(err)
		}
		if got != core.DefaultBaseURL {
			t.Errorf("got %q, want %q", got, core.DefaultBaseURL)
		}
	})

	t.Run("config file beats the default", func(t *testing.T) {
		writeConfig(t, "base-url: "+fromConfig+"\n")
		t.Setenv(baseURLEnvVar, "")
		got, err := resolveBaseURL(newURLCmd(t))
		if err != nil {
			t.Fatal(err)
		}
		if got != fromConfig {
			t.Errorf("got %q, want %q", got, fromConfig)
		}
	})

	t.Run("environment beats the config file", func(t *testing.T) {
		writeConfig(t, "base-url: "+fromConfig+"\n")
		t.Setenv(baseURLEnvVar, fromEnv)
		got, err := resolveBaseURL(newURLCmd(t))
		if err != nil {
			t.Fatal(err)
		}
		if got != fromEnv {
			t.Errorf("got %q, want %q", got, fromEnv)
		}
	})

	t.Run("flag beats everything", func(t *testing.T) {
		writeConfig(t, "base-url: "+fromConfig+"\n")
		t.Setenv(baseURLEnvVar, fromEnv)
		c := newURLCmd(t)
		if err := c.Flags().Set("base-url", fromFlag); err != nil {
			t.Fatal(err)
		}
		got, err := resolveBaseURL(c)
		if err != nil {
			t.Fatal(err)
		}
		if got != fromFlag {
			t.Errorf("got %q, want %q", got, fromFlag)
		}
	})
}

// TestResolveBaseURLValidates checks that an unusable value is refused where it
// is given, rather than producing a link that goes nowhere.
func TestResolveBaseURLValidates(t *testing.T) {
	for _, bad := range []string{
		"not a url",
		"example.com/cyberchef", // no scheme
		"ftp://example.com/",    // not a web address
		"https://",              // no host
	} {
		t.Setenv(configEnvVar, filepath.Join(t.TempDir(), "absent.yaml"))
		t.Setenv(baseURLEnvVar, bad)
		if _, err := resolveBaseURL(newURLCmd(t)); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// TestResolveBaseURLNamesTheSource checks that the error says where the bad
// value came from, since three places can supply it.
func TestResolveBaseURLNamesTheSource(t *testing.T) {
	t.Setenv(configEnvVar, filepath.Join(t.TempDir(), "absent.yaml"))
	t.Setenv(baseURLEnvVar, "nonsense")
	_, err := resolveBaseURL(newURLCmd(t))
	if err == nil || !strings.Contains(err.Error(), baseURLEnvVar) {
		t.Errorf("error should name %s: %v", baseURLEnvVar, err)
	}

	writeConfig(t, "base-url: nonsense\n")
	t.Setenv(baseURLEnvVar, "")
	_, err = resolveBaseURL(newURLCmd(t))
	if err == nil || !strings.Contains(err.Error(), "config.yaml") {
		t.Errorf("error should name the config file: %v", err)
	}
}

// TestConfigPathWithNoHome covers the case where there is no home directory to
// search: there is nowhere to look, and having no config is not an error.
func TestConfigPathWithNoHome(t *testing.T) {
	t.Setenv(configEnvVar, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if got := configPath(); got != "" {
		t.Errorf("configPath() = %q, want empty", got)
	}
	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("no home directory should not be an error: %v", err)
	}
	if cfg.BaseURL != "" {
		t.Errorf("BaseURL = %q, want empty", cfg.BaseURL)
	}
}

// TestLoadConfigUnreadable covers a config path that exists but cannot be read
// as a file. Unlike a missing file, this is reported.
func TestLoadConfigUnreadable(t *testing.T) {
	t.Setenv(configEnvVar, t.TempDir()) // a directory, not a file
	if _, err := loadConfig(); err == nil {
		t.Fatal("a config path naming a directory should be an error")
	}
}

// TestResolveBaseURLReportsConfigErrors checks that a broken config file fails
// the command rather than falling through to the default.
func TestResolveBaseURLReportsConfigErrors(t *testing.T) {
	writeConfig(t, "base-url: [broken\n")
	t.Setenv(baseURLEnvVar, "")
	if _, err := resolveBaseURL(newURLCmd(t)); err == nil {
		t.Fatal("a malformed config file should fail the command")
	}
}

// TestCheckBaseURLUnparsable covers a value url.Parse itself rejects, as
// opposed to one that parses but names no web address.
func TestCheckBaseURLUnparsable(t *testing.T) {
	if _, err := checkBaseURL("http://exa\x7fmple.com/", "--base-url"); err == nil {
		t.Fatal("a value with a control character should be refused")
	}
}
