package annotations

import (
	"github.com/aws/aws-sdk-go/service/elbv2"
	"testing"
)

const clusterName = "testCluster"

func TestParseAnnotations(t *testing.T) {
	_, err := ParseAnnotations(nil, clusterName)
	if err == nil {
		t.Fatalf("ParseAnnotations should not accept nil for annotations")
	}
}

func TestSetScheme(t *testing.T) {
	var tests = []struct {
		scheme   string
		expected string
		pass     bool
	}{
		{"", "", false},
		{"internal", "internal", true},
		{"internal", "internet-facing", false},
		{"internet-facing", "internal", false},
		{"internet-facing", "internet-facing", true},
	}

	for _, tt := range tests {
		a := &Annotations{}

		err := a.setScheme(map[string]string{schemeKey: tt.scheme})
		if err != nil && tt.pass {
			t.Errorf("setScheme(%v): expected %v, errored: %v", tt.scheme, tt.expected, err)
		}
		if err == nil && tt.pass && tt.expected != *a.Scheme {
			t.Errorf("setScheme(%v): expected %v, actual %v", tt.scheme, tt.expected, *a.Scheme)
		}
		if err == nil && !tt.pass && tt.expected == *a.Scheme {
			t.Errorf("setScheme(%v): expected %v, actual %v", tt.scheme, tt.expected, *a.Scheme)
		}
	}
}

// Should fail to create due to healthchecktimeout being greater than HealthcheckIntervalSeconds
func TestHealthcheckSecondsValidation(t *testing.T) {
	a := &Annotations{}
	if err := a.setHealthcheckIntervalSeconds(map[string]string{healthcheckIntervalSecondsKey: "5"}); err != nil {
		t.Errorf("Unexpected error seting HealthcheckIntervalSeconds. Error: %s", err.Error())
	}

	if err := a.setHealthcheckTimeoutSeconds(map[string]string{healthcheckTimeoutSecondsKey: "10"}); err == nil {
		t.Errorf("Set healthchecktimeoutSeconds when it should have failed due to being higher than HealthcheckIntervalSeconds")
	}
}

// Should fail when idle timeout is not in range 1-3600. Should succeed otherwise.
func TestConnectionIdleTimeoutValidation(t *testing.T) {
	a := &Annotations{}

	err := a.setConnectionIdleTimeout(map[string]string{connectionIdleTimeoutKey: "15"})
	if err != nil || a.ConnectionIdleTimeout == 0 {
		t.Error("Failed to set connection idle timeout when value was correct.")
	}

	err = a.setConnectionIdleTimeout(map[string]string{connectionIdleTimeoutKey: "3700"})
	if err == nil {
		t.Error("Succeeded setting connection idle timeout when value was incorrect")
	}
}

func TestSetAttributesAsList(t *testing.T) {
	annotations := &Annotations{}
	expected := elbv2.LoadBalancerAttribute{}
	expected.SetKey("access_logs.s3.enabled")
	expected.SetValue("true")

	attributes := map[string]string{attributesKey: "access_logs.s3.enabled=true"}
	err := annotations.setAttributes(attributes)

	if err != nil || len(annotations.Attributes) != 1 {
		t.Errorf("setAttributes - number of attributes incorrect")
	}

	actual := annotations.Attributes[0]

	if err == nil && *actual.Key != *expected.Key || *actual.Value != *expected.Value {
		t.Errorf("setAttributes - values did not match")
	}
}

func TestSetBadAttributesAsList(t *testing.T) {
	annotations := &Annotations{}

	attributes := map[string]string{attributesKey: "access_logs.s3.enabled=true=false"}
	err := annotations.setAttributes(attributes)

	if err == nil {
		t.Errorf("setAttributes allowed invalid annotations")
	}
}
