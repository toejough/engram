package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	_ = doHTTP()
	_ = doDynamicHTTP(&client{apiURL: "https://example.com/api"})
	_ = readConfig()
	_ = writeConfig()
	_ = encodeJSON()
	_ = runGit()
	_ = readEnv()
}

// unexported constants.
const (
	apiURL = "https://api.example.com/v1/things"
)

type client struct {
	apiURL string
}

func doDynamicHTTP(c *client) error {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, c.apiURL, nil)
	_, err := http.DefaultClient.Do(req)
	return err
}

func doHTTP() error {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	_, err := http.DefaultClient.Do(req)
	return err
}

func encodeJSON() error {
	_, err := json.Marshal(map[string]string{"k": "v"})
	return err
}

func readConfig() error {
	home, _ := os.UserHomeDir()
	_, err := os.Open(filepath.Join(home, ".config", "scanrepo", "settings.toml"))
	return err
}

func readEnv() string {
	return os.Getenv("SCANREPO_API_KEY")
}

func runGit() error {
	return exec.Command("git", "status").Run()
}

func writeConfig() error {
	home, _ := os.UserHomeDir()
	return os.WriteFile(filepath.Join(home, ".local", "share", "scanrepo", "data.bin"), []byte("x"), 0o600)
}
