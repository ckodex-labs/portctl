// A generated module for GrypeModule functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/grype-module/internal/dagger"
	"fmt"
)

// GrypeModule provides Dagger functions for scanning directories with Grype for vulnerabilities.
type GrypeModule struct{}

// Returns a container that echoes whatever string argument is provided
func (m *GrypeModule) ContainerEcho(stringArg string) *dagger.Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *GrypeModule) GrepDir(ctx context.Context, directoryArg *dagger.Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}

// ScanDirectory scans the provided directory for vulnerabilities using Grype and returns the results in the specified output format (e.g., 'json', 'table').
// If outputFormat is empty, 'json' is used by default.
func (m *GrypeModule) ScanDirectory(ctx context.Context, directoryArg *dagger.Directory, outputFormat string) (string, error) {
	if outputFormat == "" {
		outputFormat = "json"
	}
	out, err := dag.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "--no-cache", "curl"}).
		WithExec([]string{"sh", "-c", "curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin"}).
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grype", "/mnt", "--output", outputFormat}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("Grype scan returned empty output")
	}
	return out, nil
}
