package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A zero-value Image (a config-only image: no base image and no layer
// entries) must marshal `"layers": []`, not `"layers": null`.
func TestImageMarshalNilLayersAsEmpty(t *testing.T) {
	var img Image
	b, err := json.Marshal(img)
	assert.NoError(t, err)
	var fields map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(b, &fields))
	assert.JSONEq(t, `[]`, string(fields["layers"]))
}

// Round-trip: normalization must not disturb populated layers.
func TestImageMarshalRoundTrip(t *testing.T) {
	img := Image{
		Version: ImageVersion,
		Layers:  []Layer{{Digest: "sha256:abc", DiffIDs: "sha256:def"}},
	}
	b, err := json.Marshal(img)
	assert.NoError(t, err)
	var back Image
	assert.NoError(t, json.Unmarshal(b, &back))
	assert.Equal(t, img.Layers, back.Layers)
}
