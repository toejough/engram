package embed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MarshalSidecar encodes s as compact JSON. Vectors are large; pretty-
// printing them wastes disk and noises downstream diffs.
//
// json.Marshal of a Sidecar (a struct of typed-string / int / []float32
// / string fields, none of which implement MarshalJSON) cannot fail —
// the encoder only errors on cyclic data or custom marshaler failures.
// We swallow the error pointer to avoid the unreachable branch
// confusing coverage tools.
func MarshalSidecar(s Sidecar) []byte {
	out, _ := json.Marshal(s) //nolint:errchkjson // embedding vectors never contain NaN/Inf

	return out
}

// SidecarPath returns the .vec.json path sibling to a note's .md path.
// Non-.md inputs get .vec.json appended unchanged (defensive).
func SidecarPath(notePath string) string {
	if !strings.HasSuffix(notePath, ".md") {
		return notePath + sidecarExt
	}

	return strings.TrimSuffix(notePath, ".md") + sidecarExt
}

// UnmarshalSidecar decodes a sidecar from JSON, returning ErrSidecarMalformed
// on parse failure or ErrDimsMismatch when len(Vector) != Dims.
func UnmarshalSidecar(data []byte) (Sidecar, error) {
	var sidecar Sidecar

	err := json.Unmarshal(data, &sidecar)
	if err != nil {
		return Sidecar{}, fmt.Errorf("%w: %w", ErrSidecarMalformed, err)
	}

	if len(sidecar.Vector) != sidecar.Dims {
		return Sidecar{}, fmt.Errorf(
			"%w: dims=%d len=%d",
			ErrDimsMismatch,
			sidecar.Dims,
			len(sidecar.Vector),
		)
	}

	return sidecar, nil
}

// unexported constants.
const (
	sidecarExt = ".vec.json"
)
