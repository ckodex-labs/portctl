package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// Ensure bin/ exists
	if err := os.MkdirAll("bin", 0750); err != nil {
		panic("Failed to create bin directory: " + err.Error())
	}
	// Build the CLI binary from the correct path
	cmd := exec.Command("go", "build", "-o", "bin/portctl", "../../cmd/portctl/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("Failed to build portctl CLI: " + err.Error())
	}

	// Prepend bin/ to $PATH
	binDir, _ := filepath.Abs("bin")
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		// TODO: handle error appropriately (log, return, etc.)
		panic("failed to set PATH: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestPortctl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Portctl CLI Suite")
}

var _ = Describe("portctl CLI", func() {
	It("should list processes", func() {
		cmd := exec.Command("portctl", "list")
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		output := string(out)
		Expect(
			strings.Contains(output, "PID") || strings.Contains(output, "No processes found matching filters"),
		).To(BeTrue(), "output: %s", output)
	})
})
