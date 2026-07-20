package cli_test

// sliceIndex returns the first index of target in sl, -1 if absent.
func sliceIndex(sl []string, target string) int {
	for i, v := range sl {
		if v == target {
			return i
		}
	}

	return -1
}
