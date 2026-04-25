package main

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const apiURL = "https://api.example.com/v1/things"

func main() {
	_ = doHTTP()
	_ = readConfig()
	_ = runGit()
	_ = readEnv()
}

func doHTTP() error {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	_, err := http.DefaultClient.Do(req)
	return err
}

func readConfig() error {
	home, _ := os.UserHomeDir()
	_, err := os.Open(filepath.Join(home, ".config", "scanrepo", "settings.toml"))
	return err
}

func runGit() error {
	return exec.Command("git", "status").Run()
}

func readEnv() string {
	return os.Getenv("SCANREPO_API_KEY")
}
