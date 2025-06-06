package steps

import (
	"github.com/cucumber/godog"
	"os/exec"
	"strings"
	"fmt"
)

var lastOutput string

func iRun(cmd string) error {
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
