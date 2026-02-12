package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestLSP_BinaryField_TOML(t *testing.T) {
	tests := []struct {
		name     string
		toml     string
		expected LSP
	}{
		{
			name: "with binary field",
			toml: `
name = "test"
flake = "nixpkgs#gopls"
binary = "gopls"
extensions = ["go"]
`,
			expected: LSP{
				Name:       "test",
				Flake:      "nixpkgs#gopls",
				Binary:     "gopls",
				Extensions: []string{"go"},
			},
		},
		{
			name: "without binary field",
			toml: `
name = "test"
flake = "nixpkgs#gopls"
extensions = ["go"]
`,
			expected: LSP{
				Name:       "test",
				Flake:      "nixpkgs#gopls",
				Binary:     "",
				Extensions: []string{"go"},
			},
		},
		{
			name: "with binary as relative path",
			toml: `
name = "test"
flake = "github:owner/repo"
binary = "bin/custom-lsp"
extensions = ["custom"]
`,
			expected: LSP{
				Name:       "test",
				Flake:      "github:owner/repo",
				Binary:     "bin/custom-lsp",
				Extensions: []string{"custom"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lsp LSP
			if err := toml.Unmarshal([]byte(tt.toml), &lsp); err != nil {
				t.Fatalf("failed to parse TOML: %v", err)
			}

			if lsp.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, lsp.Name)
			}
			if lsp.Flake != tt.expected.Flake {
				t.Errorf("Flake: expected %q, got %q", tt.expected.Flake, lsp.Flake)
			}
			if lsp.Binary != tt.expected.Binary {
				t.Errorf("Binary: expected %q, got %q", tt.expected.Binary, lsp.Binary)
			}
			if len(lsp.Extensions) != len(tt.expected.Extensions) {
				t.Errorf("Extensions length: expected %d, got %d", len(tt.expected.Extensions), len(lsp.Extensions))
			}
		})
	}
}

func TestConfig_BinaryField_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "lsps.toml")

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	originalConfig := &Config{
		Socket: "/tmp/test.sock",
		LSPs: []LSP{
			{
				Name:       "gopls",
				Flake:      "nixpkgs#gopls",
				Binary:     "gopls",
				Extensions: []string{"go"},
			},
			{
				Name:       "custom",
				Flake:      "github:owner/repo",
				Binary:     "bin/custom-lsp",
				Extensions: []string{"custom"},
			},
			{
				Name:       "default",
				Flake:      "nixpkgs#rust-analyzer",
				Extensions: []string{"rs"},
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	if err := Save(originalConfig); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(loadedConfig.LSPs) != len(originalConfig.LSPs) {
		t.Fatalf("expected %d LSPs, got %d", len(originalConfig.LSPs), len(loadedConfig.LSPs))
	}

	for i, expectedLSP := range originalConfig.LSPs {
		gotLSP := loadedConfig.LSPs[i]

		if gotLSP.Name != expectedLSP.Name {
			t.Errorf("LSP[%d] Name: expected %q, got %q", i, expectedLSP.Name, gotLSP.Name)
		}
		if gotLSP.Flake != expectedLSP.Flake {
			t.Errorf("LSP[%d] Flake: expected %q, got %q", i, expectedLSP.Flake, gotLSP.Flake)
		}
		if gotLSP.Binary != expectedLSP.Binary {
			t.Errorf("LSP[%d] Binary: expected %q, got %q", i, expectedLSP.Binary, gotLSP.Binary)
		}
	}
}

func TestConfig_BinaryOmitempty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "lux", "lsps.toml")

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	config := &Config{
		LSPs: []LSP{
			{
				Name:       "test",
				Flake:      "nixpkgs#gopls",
				Extensions: []string{"go"},
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	if err := Save(config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if contains(content, "binary") {
		t.Error("expected binary field to be omitted when empty, but it was present in TOML")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

func TestAddLSP_WithBinary(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	if err := os.MkdirAll(filepath.Join(tmpDir, "lux"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	lsp := LSP{
		Name:       "test-lsp",
		Flake:      "nixpkgs#test",
		Binary:     "custom-binary",
		Extensions: []string{"test"},
	}

	if err := AddLSP(lsp); err != nil {
		t.Fatalf("failed to add LSP: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.LSPs) != 1 {
		t.Fatalf("expected 1 LSP, got %d", len(cfg.LSPs))
	}

	if cfg.LSPs[0].Binary != "custom-binary" {
		t.Errorf("expected binary %q, got %q", "custom-binary", cfg.LSPs[0].Binary)
	}
}

func TestAddLSP_UpdateWithBinary(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	if err := os.MkdirAll(filepath.Join(tmpDir, "lux"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	firstLSP := LSP{
		Name:       "test-lsp",
		Flake:      "nixpkgs#test",
		Extensions: []string{"test"},
	}

	if err := AddLSP(firstLSP); err != nil {
		t.Fatalf("failed to add first LSP: %v", err)
	}

	updatedLSP := LSP{
		Name:       "test-lsp",
		Flake:      "nixpkgs#test-v2",
		Binary:     "custom-binary",
		Extensions: []string{"test"},
	}

	if err := AddLSP(updatedLSP); err != nil {
		t.Fatalf("failed to update LSP: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.LSPs) != 1 {
		t.Fatalf("expected 1 LSP, got %d", len(cfg.LSPs))
	}

	if cfg.LSPs[0].Binary != "custom-binary" {
		t.Errorf("expected binary %q, got %q", "custom-binary", cfg.LSPs[0].Binary)
	}
	if cfg.LSPs[0].Flake != "nixpkgs#test-v2" {
		t.Errorf("expected flake %q, got %q", "nixpkgs#test-v2", cfg.LSPs[0].Flake)
	}
}
