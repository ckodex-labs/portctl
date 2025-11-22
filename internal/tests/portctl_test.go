package tests

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// Build the portctl CLI before running tests
	buildCmd := exec.Command("go", "build", "-o", "/tmp/portctl", "../../cmd/portctl/main.go")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Failed to build portctl CLI: %v\nOutput: %s", err, output))
	}

	// Prepend /tmp to $PATH so the test can find the built binary
	if err := os.Setenv("PATH", "/tmp"+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
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
