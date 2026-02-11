package main

import (
	"fmt"

	"github.com/toejough/projctl/internal/screenshot"
)

type screenshotDiffArgs struct {
	Expected      string `targ:"flag,short=e,required,desc=Path to expected image"`
	Actual        string `targ:"flag,short=a,required,desc=Path to actual image"`
	HeatmapOutput string `targ:"flag,desc=Path to write SSIM heatmap image"`
	DiffOutput    string `targ:"flag,desc=Path to write diff image with bounding boxes"`
	Threshold     float64 `targ:"flag,desc=SSIM threshold for cluster detection (default 0.9)"`
}

func screenshotDiff(args screenshotDiffArgs) error {
	result, err := screenshot.Diff(args.Expected, args.Actual, screenshot.DiffOpts{
		HeatmapOutput: args.HeatmapOutput,
		DiffOutput:    args.DiffOutput,
		Threshold:     args.Threshold,
	}, screenshot.RealFS{})
	if err != nil {
		return err
	}

	output, err := result.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(output)

	return nil
}
