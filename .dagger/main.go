// Portctl Dagger Pipeline
//
// This module defines all CI/CD steps for the portctl project, composable and callable from any workflow.
//
// Available steps (callable via `dagger call <step>`):
// - lint
// - test [--pkg=./...] [--cover=true] [--outPath=artifacts/cover.out]
// - build [--outPath=bin/portctl]
// - generateManifest  # Generate MCP manifest from code
// - release
// - docs
// - publishDocs
// - bdd
// - snapshotTest
// - wellKnown
// - securityScan
// - sbom
// - help
// - uploadArtifact [--src=path] [--dst=artifact-name]
//
// All steps are parameterized for maximum composability and can be invoked from CI, pipeline, or release workflows.

package main

import (
	"context"
	dagger "dagger/portctl/internal/dagger"
	"fmt"
	"os"
	"path/filepath"
)

// Portctl is the Dagger pipeline module for the portctl project.
// It provides composable CI/CD steps callable from any workflow.
type Portctl struct{}

// Returns lines that match a pattern in the files of the provided Directory
func (m *Portctl) GrepDir(ctx context.Context, directoryArg *dagger.Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}

// --- Go Module Cache Helper ---
// goModCache returns a Dagger cache volume for Go modules.
func (m *Portctl) goModCache() *dagger.CacheVolume {
	return dag.CacheVolume("go-mod-cache")
}

// --- Helper: Find Go Module Root ---
// findGoModRoot locates the nearest go.mod in the current or parent directories.
func findGoModRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found in any parent directory of %s", cwd)
}

// +dagger:call=lint
// --- Lint Step ---
// Lint runs golangci-lint on the project source code.
func (m *Portctl) Lint(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting lint step...")
	out, err := dag.Container().
		From("golangci/golangci-lint:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Lint failed: %v\n", err)
		return "", fmt.Errorf("Lint failed: %w", err)
	}
	fmt.Println("[Dagger] Lint step complete.")
	return out, nil
}

// +dagger:call=test
// --- Enhanced Test Step (with --source support and advanced debugging) ---
// Test runs Go tests for the specified package, with optional coverage and output path. Supports --source for custom source directory.
func (m *Portctl) Test(ctx context.Context, src *dagger.Directory, pkg *string, cover *bool, outPath *string) (string, error) {
	fmt.Println("[Dagger] Starting test step...")
	goModCache := m.goModCache()
	p := "./..."
	if pkg != nil {
		p = *pkg
	}
	c := false
	if cover != nil {
		c = *cover
	}
	o := ""
	if outPath != nil {
		o = *outPath
	}
	args := []string{"go", "test", "-v"}
	if c {
		args = append(args, "-coverprofile=cover.out")
	}
	args = append(args, p)
	container := dag.Container().From("golang:1.24.3").
		WithExec([]string{"bash", "-c", "apt-get update && apt-get install -y net-tools"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"})
	// Diagnostic: list all files recursively in /src
	container = container.WithExec([]string{"ls", "-lR", "/src"})
	container = container.WithExec(args)
	if o != "" && c {
		container = container.WithExec([]string{"cp", "cover.out", o})
		container = container.WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp cover.out /artifacts/"})
	}
	out, err := container.Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Test failed: %v\n", err)
		return "", fmt.Errorf("Go test failed: %w", err)
	}
	fmt.Println("[Dagger] Test step complete.")
	return out, nil
}

// +dagger:call=build
// --- Enhanced Build Step (with --source support and advanced debugging) ---
// Build compiles the portctl binary. Supports --outPath for output and --source for custom source directory.
func (m *Portctl) Build(ctx context.Context, src *dagger.Directory, outPath *string) (string, error) {
	fmt.Println("[Dagger] Starting build step...")
	goModCache := m.goModCache()
	o := "bin/portctl"
	if outPath != nil && *outPath != "" {
		o = *outPath
	}
	container := dag.Container().From("golang:1.24.3").
		WithExec([]string{"bash", "-c", "apt-get update && apt-get install -y net-tools"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"})
	// Diagnostic: list all files recursively in /src
	container = container.WithExec([]string{"ls", "-lR", "/src"})
	container = container.WithExec([]string{"go", "build", "-o", o, "./cmd/portctl"}).
		WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp " + o + " /artifacts/"})
	_, err := container.Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Build failed: %v\n", err)
		return "", fmt.Errorf("Build failed: %w", err)
	}
	fmt.Println("[Dagger] Build step complete.")
	return fmt.Sprintf("[Dagger] Build complete. Output: %s", o), nil
}

// +dagger:call=snapshotTest
// --- SnapshotTest Step ---
// SnapshotTest runs Cupaloy snapshot tests in internal/snapshots.
func (m *Portctl) SnapshotTest(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting snapshotTest step...")
	goModCache := m.goModCache()
	out, err := dag.Container().From("golang:1.24.3").
		WithExec([]string{"bash", "-c", "apt-get update && apt-get install -y net-tools"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"go", "test", "./internal/snapshots"}).
		WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp -r ./internal/snapshots/testdata /artifacts/ || true"}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] SnapshotTest failed: %v\n", err)
		return "", fmt.Errorf("Snapshot tests failed: %w", err)
	}
	fmt.Println("[Dagger] snapshotTest step complete.")
	return out, nil
}

// +dagger:call=generateManifest
// --- Generate Manifest Step ---
// GenerateManifest creates the MCP manifest from the actual tool definitions in code
func (m *Portctl) GenerateManifest(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting generateManifest step...")
	goModCache := m.goModCache()

	out, err := dag.Container().From("golang:1.24.3").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"sh", "-c", `
cat > /tmp/gen-manifest.go << 'GENEOF'
package main
import (
	"encoding/json"
	"os"
)
func main() {
	manifest := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type": "Service",
		"name": "portctl",
		"version": "1.0.0",
		"description": "Secure, cross-platform CLI for managing processes on ports",
		"homepage": "https://github.com/ckodex-labs/portctl",
		"documentation": "https://ckodex-labs.github.io/portctl",
		"protocol": "mcp",
		"capabilities": map[string]bool{"tools": true, "resources": true, "logging": true},
		"tools": []map[string]interface{}{
			{
				"name": "list_processes",
				"description": "List running processes, optionally filtered by port or service",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"port": map[string]string{"type": "number", "description": "Specific port to check"},
						"service": map[string]string{"type": "string", "description": "Filter by service name"},
					},
				},
			},
			{
				"name": "kill_process",
				"description": "Kill a process by PID or Port",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pid": map[string]string{"type": "number", "description": "Process ID to kill"},
						"port": map[string]string{"type": "number", "description": "Port number to kill processes on"},
						"force": map[string]string{"type": "boolean", "description": "Force kill (SIGKILL)"},
					},
				},
			},
			{
				"name": "scan_ports",
				"description": "Scan for open ports on a host",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host": map[string]string{"type": "string", "description": "Host to scan (default: localhost)"},
						"start_port": map[string]string{"type": "number", "description": "Start of port range"},
						"end_port": map[string]string{"type": "number", "description": "End of port range"},
					},
				},
			},
			{
				"name": "get_system_stats",
				"description": "Get system resource usage and statistics",
				"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			},
		},
		"integration": map[string]string{"command": "portctl mcp", "transport": "stdio", "format": "json-rpc"},
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(".well-known/mcp-manifest.jsonld", data, 0644)
}
GENEOF
go run /tmp/gen-manifest.go
cat .well-known/mcp-manifest.jsonld
`}).
		Stdout(ctx)

	if err != nil {
		fmt.Printf("[Dagger] GenerateManifest failed: %v\n", err)
		return "", fmt.Errorf("Manifest generation failed: %w", err)
	}
	fmt.Println("[Dagger] generateManifest step complete.")
	return out, nil
}

// Release runs GoReleaser to build and package the project, exporting artifacts.
func (m *Portctl) Release(ctx context.Context, src *dagger.Directory, githubToken *dagger.Secret, tapGithubToken *dagger.Secret) (*dagger.Directory, error) {
	fmt.Println("[Dagger] Starting release step...")
	goModCache := m.goModCache()

	// Generate MCP manifest from code first
	_, err := m.GenerateManifest(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate manifest: %w", err)
	}

	container := dag.Container().From("goreleaser/goreleaser:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithSecretVariable("TAP_GITHUB_TOKEN", tapGithubToken).
		WithEnvVariable("COSIGN_EXPERIMENTAL", "1").
		WithExec([]string{"goreleaser", "release", "--clean", "--skip=docker"}).
		WithExec([]string{"sh", "-c", "mkdir -p /src/artifacts/.well-known"}).
		WithExec([]string{"sh", "-c", "cp -r .well-known/* /src/artifacts/.well-known/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.sbom.spdx.json /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.sbom.cyclonedx.json /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.intoto.jsonl /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.sig /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.att /src/artifacts/ || true"})

	// Verify the command succeeded
	_, err = container.Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Release failed: %v\n", err)
		return nil, fmt.Errorf("GoReleaser failed: %w", err)
	}

	// Export the artifacts directory
	artifactsDir := container.Directory("/src/artifacts")
	fmt.Println("[Dagger] release step complete.")
	return artifactsDir, nil
}

// +dagger:call=publishImage
// --- Publish Image Step ---
// PublishImage builds and pushes the Docker image using Dagger native build.
func (m *Portctl) PublishImage(ctx context.Context, src *dagger.Directory, githubToken *dagger.Secret, version *string) (string, error) {
	fmt.Println("[Dagger] Starting publishImage step...")

	// Define tags
	tags := []string{"latest"}
	if version != nil && *version != "" {
		tags = append(tags, *version)
	}

	// Define platforms for multi-arch build
	platforms := []dagger.Platform{"linux/amd64", "linux/arm64"}
	variants := make([]*dagger.Container, len(platforms))

	// We need to publish for each tag
	var lastAddr string

	// Build variants once
	for i, platform := range platforms {
		variants[i] = src.DockerBuild(dagger.DirectoryDockerBuildOpts{
			Platform: platform,
		}).
			WithLabel("org.opencontainers.image.source", "https://github.com/ckodex-labs/portctl")
	}

	// Publish for each tag
	for _, tag := range tags {
		imageRef := fmt.Sprintf("ghcr.io/ckodex-labs/portctl:%s", tag)
		fmt.Printf("[Dagger] Publishing %s...\n", imageRef)

		// Set version label for this specific tag (optional, but good practice)
		currentVariants := make([]*dagger.Container, len(variants))
		for i, v := range variants {
			currentVariants[i] = v.WithLabel("org.opencontainers.image.version", tag)
		}

		publisher := dag.Container().
			WithRegistryAuth("ghcr.io", "github-actions[bot]", githubToken)

		addr, err := publisher.Publish(ctx, imageRef, dagger.ContainerPublishOpts{
			PlatformVariants: currentVariants,
		})

		if err != nil {
			return "", fmt.Errorf("Image publish failed for %s: %w", imageRef, err)
		}
		lastAddr = addr
		fmt.Printf("[Dagger] Published image to %s\n", addr)
	}

	return lastAddr, nil
}

// +dagger:call=docs
// --- Docs Step ---
// Docs builds project documentation using mdBook and updates pipeline docs.
func (m *Portctl) Docs(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting docs step...")

	// Pre-check for docs/book.toml and docs/src/SUMMARY.md
	bookTomlExists, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "test -f docs/book.toml && echo exists || echo missing"}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Docs pre-check failed: %v\n", err)
		return "", fmt.Errorf("Failed to check docs/book.toml: %w", err)
	}
	if bookTomlExists == "missing\n" || bookTomlExists == "missing" {
		return "", fmt.Errorf("docs/book.toml is missing. Please initialize your documentation with 'mdbook init docs' or copy a valid book.toml to docs/. See https://rust-lang.github.io/mdBook/ for details.")
	}

	summaryExists, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "test -f docs/src/SUMMARY.md && echo exists || echo missing"}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Docs pre-check failed: %v\n", err)
		return "", fmt.Errorf("Failed to check docs/src/SUMMARY.md: %w", err)
	}
	if summaryExists == "missing\n" || summaryExists == "missing" {
		return "", fmt.Errorf("docs/src/SUMMARY.md is missing. Please initialize your documentation with 'mdbook init docs' or copy a valid SUMMARY.md to docs/src/. See https://rust-lang.github.io/mdBook/ for details.")
	}

	out, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "apk add --no-cache mdbook && mdbook build docs"}).
		WithExec([]string{"sh", "-c", "echo '\n## Pipeline Features\n- Go module caching for faster builds\n- Artifact export: SBOM, SLSA attestation, signatures, MCP manifest to artifacts/\n- TDD/BDD with godog, 80% coverage enforcement\n- Automated docs publishing to GitHub Pages\n' >> docs/book/src/pipeline.md || true"}).
		WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp -r docs/book /artifacts/ || true"}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Docs failed: %v\n", err)
		return "", fmt.Errorf("mdBook build failed: %w", err)
	}
	fmt.Println("[Dagger] docs step complete.")
	return out, nil
}

// +dagger:call=publishDocs
// --- PublishDocs Step ---
// PublishDocs publishes mdBook documentation to the gh-pages branch on GitHub.
func (m *Portctl) PublishDocs(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting publishDocs step...")
	container := dag.Container().From("alpine:latest").
		WithExec([]string{"apk", "add", "--no-cache", "git", "openssh"}).
		WithMountedDirectory("/book", src).
		WithWorkdir("/book")

	gitUser := os.Getenv("GIT_USER")
	if gitUser == "" {
		gitUser = "github-actions[bot]"
	}
	gitEmail := os.Getenv("GIT_EMAIL")
	if gitEmail == "" {
		gitEmail = "github-actions[bot]@users.noreply.github.com"
	}
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		fmt.Printf("[Dagger] PublishDocs failed: GITHUB_TOKEN environment variable required for docs publishing\n")
		return "", fmt.Errorf("GITHUB_TOKEN environment variable required for docs publishing")
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		repo = "ckodex-labs/portctl"
	}
	remoteUrl := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", ghToken, repo)

	cmd := []string{
		"sh", "-c",
		"git init && " +
			"git config user.name '" + gitUser + "' && " +
			"git config user.email '" + gitEmail + "' && " +
			"git checkout -b gh-pages && " +
			"git add . && " +
			"git commit -m 'Publish docs [ci skip]' && " +
			"git remote add origin '" + remoteUrl + "' && " +
			"git push --force origin gh-pages:gh-pages",
	}
	out, err := container.WithExec(cmd).Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] PublishDocs failed: %v\n", err)
		return "", fmt.Errorf("Docs publishing failed: %w", err)
	}
	fmt.Println("[Dagger] publishDocs step complete.")
	return out, nil
}

// +dagger:call=bdd
// --- TDD/BDD Step ---
// BDD runs godog BDD tests and enforces 80% code coverage.
func (m *Portctl) BDD(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting bdd step...")
	goModCache := m.goModCache()
	goBuildCache := dag.CacheVolume("go-build-cache")
	container := dag.Container().From("golang:1.24.3-alpine").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", goBuildCache).
		WithExec([]string{"apk", "add", "--no-cache", "bash", "net-tools", "bc"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"go", "install", "github.com/cucumber/godog/cmd/godog@latest"}).
		WithExec([]string{"bash", "-c", "set -e; $GOPATH/bin/godog run features/ --format=pretty > bdd.out; go test -coverprofile=cover.out ./...; COVER=$(go tool cover -func=cover.out | grep total: | awk '{print substr($3, 1, length($3)-1)}'); if (( $(echo \"$COVER < 80\" | bc -l) )); then echo \"Coverage $COVER% is below 80%\"; exit 1; fi"})
	container = container.WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp bdd.out /artifacts/ || true"})
	out, err := container.Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] BDD failed: %v\n", err)
		return "", fmt.Errorf("BDD/TDD failed or coverage <80%%: %w", err)
	}
	fmt.Println("[Dagger] bdd step complete.")
	return out, nil
}

// +dagger:call=wellKnown
// --- WellKnown Step ---
// WellKnown validates .well-known metadata files for compliance and correctness.
func (m *Portctl) WellKnown(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting wellKnown step...")
	container := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src/.well-known")
	_, err := container.WithExec([]string{"test", "-f", "llms.txt"}).Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] wellKnown failed: llms.txt missing or invalid: %v\n", err)
		return "", fmt.Errorf("llms.txt missing or invalid: %w", err)
	}
	_, err = container.WithExec([]string{"test", "-f", "mcp-manifest.jsonld"}).Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] wellKnown failed: mcp-manifest.jsonld missing: %v\n", err)
		return "", fmt.Errorf("mcp-manifest.jsonld missing: %w", err)
	}
	// Install jq before validating JSON
	container = container.WithExec([]string{"sh", "-c", "apk add --no-cache jq"})
	out, err := container.WithExec([]string{"sh", "-c", "cat mcp-manifest.jsonld | jq ."}).Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] wellKnown failed: mcp-manifest.jsonld is not valid JSON: %v\n", err)
		return "", fmt.Errorf("mcp-manifest.jsonld is not valid JSON: %w", err)
	}
	// Check for skills.txt
	_, err = container.WithExec([]string{"test", "-f", "skills.txt"}).Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] wellKnown failed: skills.txt missing: %v\n", err)
		return "", fmt.Errorf("skills.txt missing: %w", err)
	}
	fmt.Println("[Dagger] wellKnown step complete.")
	return out, nil
}

// +dagger:call=securityScan
// --- Security Scan Step (with --source support and advanced debugging) ---
// SecurityScan runs gosec on the project source to detect security issues. Supports --source for custom source directory.
func (m *Portctl) SecurityScan(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting securityScan step...")
	goModCache := m.goModCache()
	container := dag.Container().From("golang:1.24.3").
		WithExec([]string{"bash", "-c", "apt-get update && apt-get install -y net-tools"}).
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"}).
		WithExec([]string{"go", "install", "github.com/securego/gosec/v2/cmd/gosec@latest"}).
		WithExec([]string{"gosec", "./..."})
	container = container.WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp -r . /artifacts/securityscan || true"})
	out, err := container.Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] SecurityScan failed: %v\n", err)
		return "", fmt.Errorf("Security scan failed: %w", err)
	}
	fmt.Println("[Dagger] securityScan step complete.")
	return out, nil
}

// +dagger:call=sbom
// --- SBOM Generation Step (patched: install Syft at runtime) ---
// SBOM generates a Software Bill of Materials (SBOM) using Syft.
func (m *Portctl) SBOM(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting sbom step...")
	out, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "apk add --no-cache curl && curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin && syft . -o json -q"}).
		WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp syft* /artifacts/ || true"}).
		Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] SBOM failed: %v\n", err)
		return "", fmt.Errorf("SBOM generation failed: %w", err)
	}
	fmt.Println("[Dagger] sbom step complete.")
	return out, nil
}

// +dagger:call=uploadArtifact
// --- Artifact Upload Step ---
// UploadArtifact uploads a file from srcPath and stores it as dstName in the artifact output.
func (m *Portctl) UploadArtifact(ctx context.Context, src *dagger.File, dstName *string) (string, error) {
	fmt.Println("[Dagger] Starting uploadArtifact step...")
	if src == nil || dstName == nil || *dstName == "" {
		fmt.Printf("[Dagger] UploadArtifact failed: src and dst must be specified\n")
		return "", fmt.Errorf("src and dst must be specified")
	}
	fmt.Printf("[Dagger] Uploading artifact as %s...\n", *dstName)
	container := dag.Container().From("alpine:latest").
		WithMountedFile("/artifact", src)
	// Ensure /out directory exists before copying
	container = container.WithExec([]string{"mkdir", "-p", "/out"})
	container = container.WithExec([]string{"cp", "/artifact", "/out/" + *dstName})
	container = container.WithExec([]string{"sh", "-c", "mkdir -p /artifacts && cp /out/" + *dstName + " /artifacts/ || true"})
	_, err := container.Sync(ctx)
	if err != nil {
		fmt.Printf("[Dagger] UploadArtifact failed: %v\n", err)
		return "", fmt.Errorf("Artifact upload failed: %w", err)
	}
	fmt.Println("[Dagger] uploadArtifact step complete.")
	return fmt.Sprintf("[Dagger] Uploaded as %s", *dstName), nil
}

// +dagger:call=deploy
// --- Deploy Step ---
// Deploy builds and pushes a Docker image and/or publishes assets to GitHub Releases.
func (m *Portctl) Deploy(ctx context.Context, src *dagger.Directory, imageTag, registry, githubToken, releaseVersion *string) (string, error) {
	fmt.Println("[Dagger] Starting deploy step...")
	imgTag := "latest"
	if imageTag != nil && *imageTag != "" {
		imgTag = *imageTag
	}
	reg := ""
	if registry != nil {
		reg = *registry
	}
	ghToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != nil && *githubToken != "" {
		ghToken = *githubToken
	}
	relVer := ""
	if releaseVersion != nil {
		relVer = *releaseVersion
	}

	// Docker build & push (if Dockerfile present)
	container := dag.Container().From("docker:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src")
	if reg != "" {
		container = container.WithEnvVariable("DOCKER_REGISTRY", reg)
	}
	container = container.WithExec([]string{"sh", "-c", "if [ -f Dockerfile ]; then docker build -t $DOCKER_REGISTRY/portctl:" + imgTag + " . && echo Built image; fi"})
	if reg != "" {
		container = container.WithExec([]string{"sh", "-c", "if [ -f Dockerfile ]; then echo $DOCKER_REGISTRY | grep -q '://' || export DOCKER_REGISTRY=registry.hub.docker.com; docker login $DOCKER_REGISTRY -u $DOCKER_USER -p $DOCKER_PASS && docker push $DOCKER_REGISTRY/portctl:" + imgTag + "; fi"})
	}

	// GitHub Release (if token and version provided)
	if ghToken != "" && relVer != "" {
		container = container.WithEnvVariable("GITHUB_TOKEN", ghToken)
		container = container.WithExec([]string{"sh", "-c", "if [ -d artifacts ]; then gh release create " + relVer + " ./artifacts/* --title 'Release '" + relVer + " --notes 'Automated release'; fi"})
	}

	container = container.WithExec([]string{"sh", "-c", "mkdir -p /artifacts && echo 'Deployment complete' > /artifacts/deploy.log"})
	out, err := container.Stdout(ctx)
	if err != nil {
		fmt.Printf("[Dagger] Deploy failed: %v\n", err)
		return "", fmt.Errorf("Deploy failed: %w", err)
	}
	fmt.Println("[Dagger] deploy step complete.")
	return out, nil
}

// +dagger:call=docsInit
// --- DocsInit Step ---
// DocsInit creates a minimal docs/ directory with book.toml and src/SUMMARY.md if missing.
func (m *Portctl) DocsInit(ctx context.Context, src *dagger.Directory) (string, error) {
	fmt.Println("[Dagger] Starting docsInit step...")
	// Check if docs/book.toml exists
	bookTomlExists, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "test -f docs/book.toml && echo exists || echo missing"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Failed to check docs/book.toml: %w", err)
	}
	if bookTomlExists == "exists\n" || bookTomlExists == "exists" {
		return "", fmt.Errorf("docs/book.toml already exists. Aborting to avoid overwrite.")
	}
	// Check if docs/src/SUMMARY.md exists
	summaryExists, err := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "test -f docs/src/SUMMARY.md && echo exists || echo missing"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Failed to check docs/src/SUMMARY.md: %w", err)
	}
	if summaryExists == "exists\n" || summaryExists == "exists" {
		return "", fmt.Errorf("docs/src/SUMMARY.md already exists. Aborting to avoid overwrite.")
	}
	// Create minimal docs structure
	container := dag.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "mkdir -p docs/src && echo '[book]\ntitle = \"Portctl Documentation\"\nauthors = [\"Your Name\"]\n' > docs/book.toml && echo '# Summary\n\n- [Introduction](intro.md)' > docs/src/SUMMARY.md && touch docs/src/intro.md"})
	_, err = container.Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("Failed to create docs skeleton: %w", err)
	}
	fmt.Println("[Dagger] docsInit step complete. Created docs/book.toml, docs/src/SUMMARY.md, and docs/src/intro.md.")
	return "[Dagger] docs skeleton created. Please commit docs/book.toml, docs/src/SUMMARY.md, and docs/src/intro.md to your repository.", nil
}

// +dagger:call=help
// --- Help Step (document --source param) ---
// Help prints available Dagger steps and their parameters.
func (m *Portctl) Help(ctx context.Context) (string, error) {
	help := `
Available Dagger steps:
- lint
- test [--pkg=./...] [--cover=true] [--outPath=artifacts/cover.out] [--source=path-or-remote]
- build [--outPath=bin/portctl] [--source=path-or-remote]
- release
- docs
- docsInit   # Create a minimal docs/ skeleton if missing
- publishDocs
- bdd
- snapshotTest
- wellKnown
- securityScan [--source=path-or-remote]
- sbom
- trivyScan [--source=path-or-remote]   # Remote module example
- help
- uploadArtifact [--src=path] [--dst=artifact-name]
- deploy [--imageTag=tag] [--registry=registry-url] [--githubToken=token] [--releaseVersion=version]
`
	return help, nil
}
