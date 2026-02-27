package screenshot

import "fmt"

// DiffArgs holds arguments for the screenshot diff command.
type DiffArgs struct {
	Expected      string  `targ:"flag,short=e,required,desc=Path to expected image"`
	Actual        string  `targ:"flag,short=a,required,desc=Path to actual image"`
	HeatmapOutput string  `targ:"flag,desc=Path to write SSIM heatmap image"`
	DiffOutput    string  `targ:"flag,desc=Path to write diff image with bounding boxes"`
	Threshold     float64 `targ:"flag,desc=SSIM threshold for cluster detection (default 0.9)"`
}

// RunDiff compares two screenshots and prints the result.
func RunDiff(args DiffArgs) error {
	result, err := Diff(args.Expected, args.Actual, DiffOpts{
		HeatmapOutput: args.HeatmapOutput,
		DiffOutput:    args.DiffOutput,
		Threshold:     args.Threshold,
	}, RealFS{})
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
