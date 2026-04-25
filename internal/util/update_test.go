package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v0.0.2", "0.0.2"},
		{"v0.1.0", "0.1.0"},
		{"v1.2.3", "1.2.3"},
		{"v0.0.2-dirty", "0.0.2"},
		{"v0.0.1-15-g75060dc-dirty", "0.0.1"},
		{"v0.0.2-3-gabcdef0", "0.0.2"},
		{"0.0.2", "0.0.2"},
		{"0.1.0", "0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeVersion(tt.input))
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"patch upgrade", "0.0.1", "0.0.2", true},
		{"minor upgrade", "0.0.2", "0.1.0", true},
		{"major upgrade", "0.9.9", "1.0.0", true},
		{"equal versions", "0.0.2", "0.0.2", false},
		{"downgrade patch", "0.0.2", "0.0.1", false},
		{"downgrade minor", "0.1.0", "0.0.9", false},
		{"v prefix on current", "v0.0.1", "0.0.2", true},
		{"v prefix on latest", "0.0.1", "v0.0.2", true},
		{"v prefix on both", "v0.0.1", "v0.0.2", true},
		{"dirty current older than latest", "v0.0.1-15-g75060dc-dirty", "0.0.2", true},
		{"dirty current equal to latest", "v0.0.2-dirty", "0.0.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsNewerVersion(tt.current, tt.latest))
		})
	}
}

func TestChangelogVersionFlow(t *testing.T) {
	tests := []struct {
		name             string
		storedVersion    string
		buildVersion     string
		changelogVersion string
		changelogShows   bool
		normalizedStored string
	}{
		{
			name:             "normal upgrade",
			storedVersion:    "0.0.1",
			buildVersion:     "v0.0.2",
			changelogVersion: "0.0.2",
			changelogShows:   true,
			normalizedStored: "0.0.2",
		},
		{
			name:             "dirty build on new tag",
			storedVersion:    "0.0.1",
			buildVersion:     "v0.0.2-dirty",
			changelogVersion: "0.0.2",
			changelogShows:   true,
			normalizedStored: "0.0.2",
		},
		{
			name:             "already up to date",
			storedVersion:    "0.0.2",
			buildVersion:     "v0.0.2",
			changelogVersion: "0.0.2",
			changelogShows:   false,
			normalizedStored: "0.0.2",
		},
		{
			name:             "dev build below changelog entry",
			storedVersion:    "0.0.1",
			buildVersion:     "v0.0.1-14-gabcdef0-dirty",
			changelogVersion: "0.0.2",
			changelogShows:   true,
			normalizedStored: "0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shows := IsNewerVersion(tt.storedVersion, tt.changelogVersion)
			assert.Equal(t, tt.changelogShows, shows)

			normalized := NormalizeVersion(tt.buildVersion)
			assert.Equal(t, tt.normalizedStored, normalized)

			if tt.changelogShows && normalized >= tt.changelogVersion {
				showsAgain := IsNewerVersion(normalized, tt.changelogVersion)
				assert.False(t, showsAgain)
			}
		})
	}
}
