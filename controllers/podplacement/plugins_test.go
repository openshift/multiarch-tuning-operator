package podplacement

import (
	"fmt"
	"testing"

	baseplugins2 "github.com/openshift/multiarch-tuning-operator/apis/multiarch/common/plugins/base_plugin"
	"github.com/openshift/multiarch-tuning-operator/apis/multiarch/common/plugins/nodeaffinityscoring"
)

func TestBasePlugin_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"Enabled Plugin", true, true},
		{"Disabled Plugin", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &baseplugins2.BasePlugin{Enabled: tt.enabled}
			if plugin.IsEnabled() != tt.expected {
				t.Errorf("Expected IsEnabled() to be %v, got %v", tt.expected, plugin.IsEnabled())
			}
		})
	}
}

func TestBasePlugin_Name(t *testing.T) {
	plugin := &baseplugins2.BasePlugin{}
	if plugin.Name() != "BasePlugin" {
		t.Errorf("Expected Name() to return 'BasePlugin', got %s", plugin.Name())
	}
}

func TestNodeAffinityScoring_Validation(t *testing.T) {
	tests := []struct {
		name       string
		platforms  []nodeaffinityscoring.NodeAffinityScoringPlatformTerm
		shouldFail bool
	}{
		{
			name: "Valid Platforms",
			platforms: []nodeaffinityscoring.NodeAffinityScoringPlatformTerm{
				{Architecture: "ppc64le", Weight: 50},
				{Architecture: "amd64", Weight: 30},
			},
			shouldFail: false,
		},
		{
			name: "Invalid Architecture",
			platforms: []nodeaffinityscoring.NodeAffinityScoringPlatformTerm{
				{Architecture: "invalid_arch", Weight: 20},
			},
			shouldFail: true,
		},
		{
			name: "Invalid Weight",
			platforms: []nodeaffinityscoring.NodeAffinityScoringPlatformTerm{
				{Architecture: "amd64", Weight: -10},
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scoring := &nodeaffinityscoring.NodeAffinityScoring{
				BasePlugin: baseplugins2.BasePlugin{Enabled: true},
				Platforms:  tt.platforms,
			}

			err := validateNodeAffinityScoring(scoring)
			if (err != nil) != tt.shouldFail {
				t.Errorf("Expected failure: %v, got error: %v", tt.shouldFail, err)
			}
		})
	}
}

func validateNodeAffinityScoring(scoring *nodeaffinityscoring.NodeAffinityScoring) error {
	for _, platform := range scoring.Platforms {
		// Check if Architecture is non-empty.
		if len(platform.Architecture) == 0 {
			return fmt.Errorf("Architecture cannot be empty")
		}
		// Check if Weight is within range.
		if platform.Weight < 0 || platform.Weight > 100 {
			return fmt.Errorf("Weight must be between 0 and 100")
		}
		// Validate architecture value (simulate Enum validation).
		validArchitectures := map[string]bool{
			"arm64":   true,
			"amd64":   true,
			"ppc64le": true,
			"s390x":   true,
		}
		if !validArchitectures[platform.Architecture] {
			return fmt.Errorf("Invalid architecture: %s", platform.Architecture)
		}
	}
	return nil
}
