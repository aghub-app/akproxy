package main

import "testing"

func TestBuildInfoExposesMetadataOnlyInDevelopment(t *testing.T) {
	previousVersion, previousRevision := version, buildRevision
	t.Cleanup(func() {
		version, buildRevision = previousVersion, previousRevision
	})

	version, buildRevision = "dev", "0123456789abcdef"
	devInfo := (&App{}).BuildInfo()
	if devInfo.Version != version || devInfo.Revision != buildRevision {
		t.Fatalf("development BuildInfo() = %+v", devInfo)
	}

	version = "0.1.4"
	if got := (&App{}).BuildInfo(); got != (BuildInfo{Version: "0.1.4"}) {
		t.Fatalf("release BuildInfo() = %+v, want version only", got)
	}
}

func TestDarkForTheme(t *testing.T) {
	for _, test := range []struct {
		theme      string
		systemDark bool
		want       bool
	}{
		{"system", true, true},
		{"system", false, false},
		{"light", true, false},
		{"dark", false, true},
	} {
		if got := darkForTheme(test.theme, test.systemDark); got != test.want {
			t.Errorf("darkForTheme(%q, %t) = %t, want %t", test.theme, test.systemDark, got, test.want)
		}
	}
}
