package steps

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cucumber/godog"
)

var lastOutput string

func iRun(cmd string) error {
	// Security: reject dangerous shell metacharacters and empty commands
	if strings.TrimSpace(cmd) == "" {
		return fmt.Errorf("command must not be empty")
	}
	if strings.ContainsAny(cmd, ";&|><`$") {
		return fmt.Errorf("command contains forbidden shell metacharacters")
	}
	parts := strings.Split(cmd, " ")
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	lastOutput = string(out)
	return err
}

func iShouldSee(expected string) error {
	if !strings.Contains(lastOutput, expected) {
		return fmt.Errorf("expected output to contain %q, got %q", expected, lastOutput)
	}
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^I run "(.*)"$`, iRun)
	ctx.Step(`^I should see "(.*)"$`, iShouldSee)
}
