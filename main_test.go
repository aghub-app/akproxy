package main

import "testing"

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
