package cli

import (
	"strings"
	"testing"

	"github.com/glincker/stacklit/internal/schema"
	"github.com/spf13/cobra"
)

func TestIndexInputFlags(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		wantDefault string
	}{
		{name: "view", cmd: viewCmd, wantDefault: defaultIndexPath},
		{name: "diff", cmd: newDiffCmd(), wantDefault: ""},
		{name: "derive", cmd: newDeriveCmd(), wantDefault: defaultIndexPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup("input")
			if flag == nil {
				t.Fatal("expected input flag to be registered")
			}
			if flag.Shorthand != "i" {
				t.Fatalf("expected -i shorthand, got %q", flag.Shorthand)
			}
			if flag.DefValue != tt.wantDefault {
				t.Fatalf("expected default %q, got %q", tt.wantDefault, flag.DefValue)
			}
		})
	}
}

func TestRemovedCommandsAreNotRegistered(t *testing.T) {
	removed := map[string]bool{
		"completion": true,
		"export":     true,
		"generate":   true,
		"init":       true,
		"serve":      true,
		"setup":      true,
	}

	for _, cmd := range rootCmd.Commands() {
		if removed[cmd.Name()] {
			t.Fatalf("command %q should not be registered", cmd.Name())
		}
	}
}

func TestDeriveDoesNotExposeInjectFlag(t *testing.T) {
	cmd := newDeriveCmd()
	if flag := cmd.Flags().Lookup("inject"); flag != nil {
		t.Fatal("derive should not expose --inject")
	}
}

func TestFindModulesSortedAndLimited(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"zeta/api":    {Purpose: "API handlers"},
			"alpha/api":   {Purpose: "API handlers"},
			"beta/api":    {Purpose: "API handlers"},
			"gamma/api":   {Purpose: "API handlers"},
			"delta/api":   {Purpose: "API handlers"},
			"epsilon/api": {Purpose: "API handlers"},
		},
	}

	got := findModules(idx, "api")
	if len(got) != 5 {
		t.Fatalf("expected 5 results, got %d", len(got))
	}

	want := []string{"alpha/api", "beta/api", "delta/api", "epsilon/api", "gamma/api"}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("result %d: expected %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestFindModulesMatchesPurposeCaseInsensitive(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/auth": {Purpose: "Authentication and authorization"},
			"internal/db":   {Purpose: "Database access"},
		},
	}

	got := findModules(idx, "AUTHORIZATION")
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Name != "internal/auth" {
		t.Fatalf("expected internal/auth, got %q", got[0].Name)
	}
}

func TestDependencyEdgesReturnsEmptyForKnownIsolatedModule(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/isolated": {Purpose: "Isolated module"},
		},
	}

	edges, err := dependencyEdges(idx, "internal/isolated")
	if err != nil {
		t.Fatalf("dependencyEdges returned error: %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("expected no edges, got %v", edges)
	}
}

func TestDependencyEdgesErrorsForUnknownModule(t *testing.T) {
	idx := &schema.Index{
		Modules: map[string]schema.ModuleInfo{
			"internal/known": {Purpose: "Known module"},
		},
	}

	_, err := dependencyEdges(idx, "internal/missing")
	if err == nil {
		t.Fatal("expected error for missing module")
	}
	if !strings.Contains(err.Error(), `module "internal/missing" not found`) {
		t.Fatalf("expected missing module error, got %v", err)
	}
}
