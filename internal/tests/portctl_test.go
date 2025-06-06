package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// Ensure bin/ exists
	if err := os.MkdirAll("bin", 0755); err != nil {
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
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

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
		Expect(string(out)).To(ContainSubstring("PID"))
	})
})
