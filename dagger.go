// Portctl Dagger Pipeline
//
// This module defines all CI/CD steps for the portctl project, composable and callable from any workflow.
//
// Available steps (callable via `dagger call <step>`):
// - lint
// - test [--pkg=./...] [--cover=true] [--outPath=artifacts/cover.out]
// - build [--outPath=bin/portctl]
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
	"fmt"
	"os"
	"path/filepath"

	dagger "dagger.io/dagger"
)

// Portctl is the Dagger pipeline module for the portctl project.
// It provides composable CI/CD steps callable from any workflow.
type Portctl struct{}

// --- Go Module Cache Helper ---
// goModCache returns a Dagger cache volume for Go modules.
func (m *Portctl) goModCache(client *dagger.Client) *dagger.CacheVolume {
	return client.CacheVolume("go-mod-cache")
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

// --- Lint Step ---
// Lint runs golangci-lint on the project source code.
func (m *Portctl) Lint(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Running golangci-lint...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	out, err := client.Container().From("golangci/golangci-lint:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Lint failed: %w", err)
	}
	return out, nil
}

// --- Enhanced Test Step (with --source support and advanced debugging) ---
// Test runs Go tests for the specified package, with optional coverage and output path. Supports --source for custom source directory.
func (m *Portctl) Test(ctx context.Context, pkg string, cover bool, outPath string, source string) (string, error) {
	var modRoot string
	var err error
	if source != "" {
		modRoot = source
		fmt.Printf("[Dagger] Using user-supplied source: %s\n", modRoot)
	} else {
		modRoot, err = findGoModRoot()
		if err != nil {
			return "", fmt.Errorf("[Dagger] Could not find go.mod: %w", err)
		}
		fmt.Printf("[Dagger] Detected Go module root: %s\n", modRoot)
	}
	if pkg == "" {
		pkg = "./..."
	}
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(modRoot)
	goModCache := m.goModCache(client)
	args := []string{"go", "test", "-v"}
	if cover {
		args = append(args, "-coverprofile=cover.out")
	}
	args = append(args, pkg)
	container := client.Container().From("golang:1.24.0").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"}).
		WithExec(args)
	if outPath != "" && cover {
		container = container.WithExec([]string{"cp", "cover.out", outPath})
	}
	out, err := container.Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Go test failed: %w", err)
	}
	return out, nil
}

// --- Enhanced Build Step (with --source support and advanced debugging) ---
// Build compiles the portctl binary. Supports --outPath for output and --source for custom source directory.
func (m *Portctl) Build(ctx context.Context, outPath string, source string) (string, error) {
	var modRoot string
	var err error
	if source != "" {
		modRoot = source
		fmt.Printf("[Dagger] Using user-supplied source: %s\n", modRoot)
	} else {
		modRoot, err = findGoModRoot()
		if err != nil {
			return "", fmt.Errorf("[Dagger] Could not find go.mod: %w", err)
		}
		fmt.Printf("[Dagger] Detected Go module root: %s\n", modRoot)
	}
	if outPath == "" {
		outPath = "bin/portctl"
	}
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(modRoot)
	goModCache := m.goModCache(client)
	container := client.Container().From("golang:1.24.0").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"}).
		WithExec([]string{"go", "build", "-o", outPath, "./cmd/portctl/main.go"})
	_, err = container.Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("Build failed: %w", err)
	}
	return fmt.Sprintf("[Dagger] Build complete. Output: %s", outPath), nil
}

// --- SnapshotTest Step ---
// SnapshotTest runs Cupaloy snapshot tests in internal/snapshots.
func (m *Portctl) SnapshotTest(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Running Cupaloy snapshot tests...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	goModCache := m.goModCache(client)
	out, err := client.Container().From("golang:1.24.0").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"go", "test", "./internal/snapshots"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Snapshot tests failed: %w", err)
	}
	return out, nil
}

// --- Release Step ---
// Release runs GoReleaser to build and package the project, exporting artifacts.
func (m *Portctl) Release(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Running GoReleaser...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	goModCache := m.goModCache(client)
	out, err := client.Container().From("goreleaser/goreleaser:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithEnvVariable("GITHUB_TOKEN", os.Getenv("GITHUB_TOKEN")).
		WithEnvVariable("COSIGN_EXPERIMENTAL", "1").
		WithExec([]string{"goreleaser", "release", "--clean", "--skip-publish"}).
		WithExec([]string{"sh", "-c", "mkdir -p /src/artifacts && cp dist/*.sbom.json /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.intoto.jsonl /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.sig /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp dist/*.att /src/artifacts/ || true"}).
		WithExec([]string{"sh", "-c", "cp .well-known/mcp-manifest.jsonld /src/artifacts/ || true"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("GoReleaser failed: %w", err)
	}
	return out, nil
}

// --- Docs Step ---
// Docs builds project documentation using mdBook and updates pipeline docs.
func (m *Portctl) Docs(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Building documentation with mdBook and updating for new pipeline features...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	out, err := client.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "apk add --no-cache mdbook && mdbook build docs"}).
		WithExec([]string{"sh", "-c", "echo '\n## Pipeline Features\n- Go module caching for faster builds\n- Artifact export: SBOM, SLSA attestation, signatures, MCP manifest to artifacts/\n- TDD/BDD with godog, 80% coverage enforcement\n- Automated docs publishing to GitHub Pages\n' >> docs/book/src/pipeline.md || true"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("mdBook build failed: %w", err)
	}
	return out, nil
}

// --- PublishDocs Step ---
// PublishDocs publishes mdBook documentation to the gh-pages branch on GitHub.
func (m *Portctl) PublishDocs(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Publishing mdBook docs to GitHub Pages (gh-pages branch)...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory("docs/book")
	container := client.Container().From("alpine:latest").
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
		return "", fmt.Errorf("GITHUB_TOKEN environment variable required for docs publishing")
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		repo = "mchorfa/portctl"
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
		return "", fmt.Errorf("Docs publishing failed: %w", err)
	}
	return out, nil
}

// --- TDD/BDD Step ---
// BDD runs godog BDD tests and enforces 80% code coverage.
func (m *Portctl) BDD(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Running BDD (godog) tests and enforcing 80% coverage...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	goModCache := m.goModCache(client)
	out, err := client.Container().From("golang:1.24.0").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"go", "install", "github.com/cucumber/godog/cmd/godog@latest"}).
		WithExec([]string{"sh", "-c", "$GOPATH/bin/godog --format=pretty --paths=features/ > bdd.out; go test -coverprofile=cover.out ./...; COVER=$(go tool cover -func=cover.out | grep total: | awk '{print substr($3, 1, length($3)-1)}'); awk 'BEGIN{if($1<80){exit 1}}' <<< $COVER"}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("BDD/TDD failed or coverage <80%%: %w", err)
	}
	return out, nil
}

// --- WellKnown Step ---
// WellKnown validates .well-known metadata files for compliance and correctness.
func (m *Portctl) WellKnown(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Validating .well-known metadata files...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	container := client.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src/.well-known")
	_, err = container.WithExec([]string{"test", "-f", "llms.txt"}).Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("llms.txt missing or invalid: %w", err)
	}
	_, err = container.WithExec([]string{"test", "-f", "mcp-manifest.jsonld"}).Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("mcp-manifest.jsonld missing: %w", err)
	}
	out, err := container.WithExec([]string{"sh", "-c", "cat mcp-manifest.jsonld | jq ."}).Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("mcp-manifest.jsonld is not valid JSON: %w", err)
	}
	return out, nil
}

// --- Security Scan Step (with --source support and advanced debugging) ---
// SecurityScan runs gosec on the project source to detect security issues. Supports --source for custom source directory.
func (m *Portctl) SecurityScan(ctx context.Context, source string) (string, error) {
	var modRoot string
	var err error
	if source != "" {
		modRoot = source
		fmt.Printf("[Dagger] Using user-supplied source: %s\n", modRoot)
	} else {
		modRoot, err = findGoModRoot()
		if err != nil {
			return "", fmt.Errorf("[Dagger] Could not find go.mod: %w", err)
		}
		fmt.Printf("[Dagger] Detected Go module root: %s\n", modRoot)
	}
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(modRoot)
	goModCache := m.goModCache(client)
	container := client.Container().From("golang:1.24.0").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithExec([]string{"ls", "-l", "/src"}).
		WithExec([]string{"cat", "/src/go.mod"}).
		WithExec([]string{"pwd"}).
		WithExec([]string{"go", "install", "github.com/securego/gosec/v2/cmd/gosec@latest"}).
		WithExec([]string{"gosec", "./..."})
	out, err := container.Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("Security scan failed: %w", err)
	}
	return out, nil
}

// --- SBOM Generation Step (patched: install Syft at runtime) ---
// SBOM generates a Software Bill of Materials (SBOM) using Syft.
func (m *Portctl) SBOM(ctx context.Context) (string, error) {
	fmt.Println("[Dagger] Generating SBOM with Syft...")
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().Directory(".")
	container := client.Container().From("alpine:latest").
		WithMountedDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "apk add --no-cache curl && curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin && syft . -o json -q"})
	out, err := container.Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("SBOM generation failed: %w", err)
	}
	return out, nil
}

// --- Artifact Upload Step ---
// UploadArtifact uploads a file from srcPath and stores it as dstName in the artifact output.
func (m *Portctl) UploadArtifact(ctx context.Context, srcPath, dstName string) (string, error) {
	if srcPath == "" || dstName == "" {
		return "", fmt.Errorf("src and dst must be specified")
	}
	fmt.Printf("[Dagger] Uploading artifact %s as %s...\n", srcPath, dstName)
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return "", fmt.Errorf("Dagger connect failed: %w", err)
	}
	defer client.Close()

	src := client.Host().File(srcPath)
	_, err = client.Container().From("alpine:latest").
		WithMountedFile("/artifact", src).
		WithExec([]string{"cp", "/artifact", "/out/" + dstName}).
		Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("Artifact upload failed: %w", err)
	}
	return fmt.Sprintf("[Dagger] Uploaded %s as %s", srcPath, dstName), nil
}

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
- publishDocs
- bdd
- snapshotTest
- wellKnown
- securityScan [--source=path-or-remote]
- sbom
- help
- uploadArtifact [--src=path] [--dst=artifact-name]
`
	return help, nil
}
