package tableimport

import (
	"bambuscribe/desktop/internal/storage"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func LoadConfig(dir string) (Config, error) {
	c := Config{BaseURL: "https://api.openai.com/v1", Protocol: "responses"}
	b, err := os.ReadFile(filepath.Join(dir, "table-agent.json"))
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}
func SaveConfig(dir string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	old, err := LoadConfig(dir)
	if err != nil {
		return err
	}
	if c.APIKey == "" {
		c.APIKey = old.APIKey
	}
	c.HasKey = false
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return storage.WriteAtomic(filepath.Join(dir, "table-agent.json"), b, 0600)
}
func Redacted(c Config) Config { c.HasKey = c.APIKey != ""; c.APIKey = ""; return c }
