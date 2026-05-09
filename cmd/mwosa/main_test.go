package main

import (
	"runtime/debug"
	"testing"
)

func TestBuildInfoFromUsesModuleVersion(t *testing.T) {
	got := buildInfoFrom("", "", "", &debug.BuildInfo{
		Main: debug.Module{Version: "v0.1.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-05-03T00:00:00Z"},
		},
	}, true)

	if got.Version != "v0.1.0" {
		t.Fatalf("Version = %q, want v0.1.0", got.Version)
	}
	if got.Commit != "abc123" {
		t.Fatalf("Commit = %q, want abc123", got.Commit)
	}
	if got.Date != "2026-05-03T00:00:00Z" {
		t.Fatalf("Date = %q, want 2026-05-03T00:00:00Z", got.Date)
	}
}

func TestBuildInfoFromKeepsInjectedValues(t *testing.T) {
	got := buildInfoFrom("v0.2.0", "override", "2026-05-04", &debug.BuildInfo{
		Main: debug.Module{Version: "v0.1.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-05-03T00:00:00Z"},
		},
	}, true)

	if got.Version != "v0.2.0" {
		t.Fatalf("Version = %q, want v0.2.0", got.Version)
	}
	if got.Commit != "override" {
		t.Fatalf("Commit = %q, want override", got.Commit)
	}
	if got.Date != "2026-05-04" {
		t.Fatalf("Date = %q, want 2026-05-04", got.Date)
	}
}

func TestBuildInfoFromIgnoresDevelopmentVersion(t *testing.T) {
	got := buildInfoFrom("", "", "", &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
	}, true)

	if got.Version != "" {
		t.Fatalf("Version = %q, want empty", got.Version)
	}
	if !got.Development {
		t.Fatal("Development = false, want true for devel build")
	}
}

func TestBuildInfoFromInjectedVersionIsNotDevelopment(t *testing.T) {
	got := buildInfoFrom("v0.2.0", "", "", &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
	}, true)

	if got.Development {
		t.Fatal("Development = true, want false when version is injected")
	}
}
