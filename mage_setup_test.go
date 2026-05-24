//go:build mage

package main

import "testing"

func TestAppNameSuggestionFromTargetDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "relative dir", in: "../my-app", want: "My App"},
		{name: "absolute dir", in: "/tmp/Customer Portal", want: "Customer Portal"},
		{name: "underscores", in: "./internal_tool", want: "Internal Tool"},
		{name: "trailing slash", in: "/tmp/roadmap-demo/", want: "Roadmap Demo"},
		{name: "dot", in: ".", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appNameSuggestionFromTargetDir(tt.in); got != tt.want {
				t.Fatalf("appNameSuggestionFromTargetDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestModulePathSuggestion(t *testing.T) {
	tests := []struct {
		name              string
		appName           string
		appNameSuggestion string
		want              string
	}{
		{name: "app name wins", appName: "Customer Portal", appNameSuggestion: "Ignored Suggestion", want: "github.com/you/customer-portal"},
		{name: "fallback to suggestion", appName: "", appNameSuggestion: "My App", want: "github.com/you/my-app"},
		{name: "trim whitespace", appName: "  Demo App  ", appNameSuggestion: "", want: "github.com/you/demo-app"},
		{name: "default fallback", appName: "", appNameSuggestion: "", want: "github.com/you/my-app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modulePathSuggestion(tt.appName, tt.appNameSuggestion); got != tt.want {
				t.Fatalf("modulePathSuggestion(%q, %q) = %q, want %q", tt.appName, tt.appNameSuggestion, got, tt.want)
			}
		})
	}
}
