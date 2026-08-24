// Package kernel holds only this boundary test. The real Kernel
// implementation packages are internal/kernel/workflow,
// internal/kernel/objective, and internal/kernel/knowledge — this file has
// no production code of its own.
package kernel

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// modulePrefix is this module's import path prefix. Every internal/...
// dependency this test discovers is reported relative to it.
const modulePrefix = "github.com/Node-Features/company-os/apps/companyd/"

// allowedInternalPrefixes are the only internal/... import prefixes a
// Kernel package may depend on, transitively. domain/* is the Kernel's
// organizational vocabulary (docs/architecture/kernel.md: "organization
// identity... objective identity... capability identity..."); fixtures is
// bootstrap data the Kernel reads through Registry; kernel/* is the Kernel
// itself.
//
// Everything else under internal/... — ports, adapters, application,
// runtime, daemon, governance, identity, intelligence, observability,
// departments, agent, concurrency — is infrastructure, orchestration, or a
// sibling layer. docs/architecture/kernel.md's Non-responsibilities says
// the Kernel "cannot depend on their concrete implementations": model
// inference, provider APIs, database drivers, transactions, event brokers,
// notifications, metrics exporters, and authoritative-state loading all
// live in internal/ports (and its implementations), so internal/ports
// itself is exactly as forbidden as any adapter.
var allowedInternalPrefixes = []string{
	modulePrefix + "internal/domain/",
	modulePrefix + "internal/fixtures",
	modulePrefix + "internal/kernel", // covers internal/kernel itself (this package) and every internal/kernel/* subpackage
}

// TestKernelPackagesDoNotImportInfrastructure discovers every package
// under internal/kernel/... and checks its full transitive dependency
// graph against allowedInternalPrefixes. It fails on any edge outside that
// allowlist — not just direct imports — because the historical version of
// this violation was transitive: internal/kernel/workflow imported
// internal/fixtures, and internal/fixtures.NewRegistryFromDB imported
// internal/ports (which declares ProviderAdapter, Authenticator, Notifier,
// and every persistence repository) purely to support a database loader
// that only cmd/companyd/main.go ever called. Moving that loader out of
// internal/fixtures (main.go now calls orgRepo.GetOrganization directly
// and applies fixtures.Registry.WithOrganization) fixed it; this test is
// what keeps it fixed.
func TestKernelPackagesDoNotImportInfrastructure(t *testing.T) {
	kernelPkgs := goList(t, modulePrefix+"internal/kernel/...")
	if len(kernelPkgs) < 3 {
		t.Fatalf("expected to discover at least 3 packages under internal/kernel/... (workflow, objective, knowledge), "+
			"got %d: %v — is `go list` misconfigured, or did a Kernel package move?", len(kernelPkgs), kernelPkgs)
	}

	for _, pkg := range kernelPkgs {
		pkg := pkg
		t.Run(strings.TrimPrefix(pkg, modulePrefix), func(t *testing.T) {
			for _, dep := range goListDeps(t, pkg) {
				if !strings.HasPrefix(dep, modulePrefix) {
					continue // stdlib or third-party module — not a layering concern here.
				}
				if !allowedInternal(dep) {
					t.Errorf("%s transitively imports %s, which is outside the Kernel's allowed boundary\n"+
						"(internal/domain/*, internal/fixtures, internal/kernel/*).\n"+
						"See docs/architecture/kernel.md's Non-responsibilities. If this dependency is genuinely\n"+
						"required, that doc's contract is changing and allowedInternalPrefixes in this test must\n"+
						"change with it, deliberately — not silently pass because the allowlist grew to match.",
						pkg, dep)
				}
			}
		})
	}
}

func allowedInternal(importPath string) bool {
	for _, p := range allowedInternalPrefixes {
		if strings.HasPrefix(importPath, p) {
			return true
		}
	}
	return false
}

// goList runs `go list <pattern>` and returns one import path per line —
// used to discover the current set of Kernel packages without hardcoding
// them, so a new internal/kernel/<x> package is covered by this test the
// day it's added.
func goList(t *testing.T, pattern string) []string {
	t.Helper()
	return runGoList(t, "list", pattern)
}

// goListDeps runs `go list -deps <pkg>` and returns the full transitive
// dependency closure, one import path per line (stdlib, third-party, and
// same-module packages all included) — exactly the set that must build
// before pkg builds.
func goListDeps(t *testing.T, pkg string) []string {
	t.Helper()
	return runGoList(t, "list", "-deps", pkg)
}

func runGoList(t *testing.T, args ...string) []string {
	t.Helper()
	cmd := exec.Command("go", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}
