package main

import "github.com/toejough/projctl/internal/screenshot"

func screenshotDiff(args screenshot.DiffArgs) error {
	return screenshot.RunDiff(args)
}
