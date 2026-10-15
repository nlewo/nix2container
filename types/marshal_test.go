package types

import (
	"encoding/json"
	"testing"
	"time"

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

func TestHealthConfigUnmarshalDurations(t *testing.T) {
	want := HealthConfig{
		Test:          []string{"CMD-SHELL", "valkey-cli ping"},
		Interval:      5 * time.Second,
		Timeout:       3 * time.Second,
		StartPeriod:   10 * time.Second,
		StartInterval: 500 * time.Millisecond,
		Retries:       3,
	}

	t.Run("nanoseconds", func(t *testing.T) {
		var got HealthConfig
		err := json.Unmarshal([]byte(`{
			"Test": ["CMD-SHELL", "valkey-cli ping"],
			"Interval": 5000000000,
			"Timeout": 3000000000,
			"StartPeriod": 10000000000,
			"StartInterval": 500000000,
			"Retries": 3
		}`), &got)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("duration strings", func(t *testing.T) {
		var got HealthConfig
		err := json.Unmarshal([]byte(`{
			"Test": ["CMD-SHELL", "valkey-cli ping"],
			"Interval": "5s",
			"Timeout": "3s",
			"StartPeriod": "10s",
			"StartInterval": "500ms",
			"Retries": 3
		}`), &got)
		assert.NoError(t, err)
		assert.Equal(t, want, got)

		b, err := json.Marshal(got)
		assert.NoError(t, err)
		assert.JSONEq(t, `{
			"Test": ["CMD-SHELL", "valkey-cli ping"],
			"Interval": 5000000000,
			"Timeout": 3000000000,
			"StartPeriod": 10000000000,
			"StartInterval": 500000000,
			"Retries": 3
		}`, string(b))
	})

	t.Run("invalid duration", func(t *testing.T) {
		var got HealthConfig
		err := json.Unmarshal([]byte(`{"Interval": "not-a-duration"}`), &got)
		assert.Error(t, err)
	})
}
