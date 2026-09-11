package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (int, error, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code, err := Run(args, &stdout, &stderr)
	return code, err, stdout.String(), stderr.String()
}

func TestRunWithoutArgumentsPrintsTheUsage(t *testing.T) {
	code, err, stdout, _ := run(t)

	if code != exitUsage || err != nil {
		t.Fatalf("code = %d, err = %v; want %d and no error", code, err, exitUsage)
	}
	if !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "install-node-and-npm") {
		t.Errorf("usage does not list the goals:\n%s", stdout)
	}
}

func TestRunPrintsHelpAndVersion(t *testing.T) {
	for _, argument := range []string{"-h", "--help", "help"} {
		code, err, stdout, _ := run(t, argument)
		if code != exitOK || err != nil {
			t.Errorf("%s: code = %d, err = %v; want 0 and no error", argument, code, err)
		}
		if !strings.Contains(stdout, "Goals:") {
			t.Errorf("%s did not print the goal list:\n%s", argument, stdout)
		}
	}
	for _, argument := range []string{"-v", "--version", "version"} {
		code, err, stdout, _ := run(t, argument)
		if code != exitOK || err != nil {
			t.Errorf("%s: code = %d, err = %v; want 0 and no error", argument, code, err)
		}
		if !strings.Contains(stdout, "frontend "+Version) {
			t.Errorf("%s did not print the version:\n%s", argument, stdout)
		}
	}
}

func TestRunRejectsAnUnknownGoal(t *testing.T) {
	code, err, _, stderr := run(t, "no-such-goal")

	if code != exitUsage {
		t.Errorf("code = %d, want %d", code, exitUsage)
	}
	if err == nil || !strings.Contains(err.Error(), `unknown goal "no-such-goal"`) {
		t.Errorf("error = %v, want it to name the goal", err)
	}
	if !strings.Contains(stderr, "Goals:") {
		t.Errorf("the usage was not written to stderr:\n%s", stderr)
	}
}

func TestRunRejectsAMissingRequiredParameter(t *testing.T) {
	code, err, _, _ := run(t, "install-node-and-yarn")

	if code != exitUsage {
		t.Errorf("code = %d, want %d", code, exitUsage)
	}
	if err == nil || err.Error() != "goal install-node-and-yarn requires --nodeVersion and --yarnVersion" {
		t.Errorf("error = %v, want it to name both parameters", err)
	}
}

func TestRunRejectsAnUnknownOption(t *testing.T) {
	code, err, _, _ := run(t, "npm", "--no-such-option")

	if code != exitUsage || err != nil {
		t.Errorf("code = %d, err = %v; want %d and no error", code, err, exitUsage)
	}
}

func TestRunRejectsASurplusArgument(t *testing.T) {
	code, err, _, _ := run(t, "npm", "--skip.npm", "surplus")

	if code != exitUsage {
		t.Errorf("code = %d, want %d", code, exitUsage)
	}
	if err == nil || !strings.Contains(err.Error(), `unexpected argument "surplus"`) {
		t.Errorf("error = %v, want it to name the argument", err)
	}
}

func TestRunPrintsAGoalsOwnHelp(t *testing.T) {
	code, err, _, stderr := run(t, "npm", "--help")

	if code != exitOK || err != nil {
		t.Fatalf("code = %d, err = %v; want 0 and no error", code, err)
	}
	if !strings.Contains(stderr, "frontend.npm.arguments") {
		t.Errorf("the goal's options were not printed:\n%s", stderr)
	}
}

func TestRunSkipsAGoalAndSucceeds(t *testing.T) {
	code, err, stdout, _ := run(t, "npm", "--skip.npm", "--settings", filepath.Join(t.TempDir(), "absent.xml"))

	if code != exitOK || err != nil {
		t.Fatalf("code = %d, err = %v; want 0 and no error", code, err)
	}
	if !strings.Contains(stdout, "Skipping execution.") {
		t.Errorf("the skip was not logged:\n%s", stdout)
	}
}

func TestRunReportsAGoalFailure(t *testing.T) {
	directory := t.TempDir()

	code, err, _, _ := run(t, "npm",
		"--workingDirectory", directory,
		"--settings", filepath.Join(directory, "absent.xml"))

	if code != exitFailure {
		t.Errorf("code = %d, want %d", code, exitFailure)
	}
	if err == nil || err.Error() != "Failed to run task" {
		t.Errorf("error = %v, want %q", err, "Failed to run task")
	}
}

func TestRunReportsUnreadableSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.xml")
	if err := os.WriteFile(path, []byte("<settings>"), 0o666); err != nil {
		t.Fatal(err)
	}

	code, err, _, _ := run(t, "npm", "--settings", path)

	if code != exitFailure {
		t.Errorf("code = %d, want %d", code, exitFailure)
	}
	if err == nil || !strings.Contains(err.Error(), "could not read "+path) {
		t.Errorf("error = %v, want it to name the settings file", err)
	}
}

func TestRunPassesSystemPropertiesToTheGoal(t *testing.T) {
	properties, remaining := extractSystemProperties([]string{
		"npm", "-DnpmRegistryURL=http://override", "-Dflag", "--skip.npm", "-D",
	})

	if properties["npmRegistryURL"] != "http://override" {
		t.Errorf("properties = %v, want the registry override", properties)
	}
	if properties["flag"] != "true" {
		t.Errorf("properties = %v, want a bare -D to set true", properties)
	}
	want := []string{"npm", "--skip.npm", "-D"}
	if strings.Join(remaining, " ") != strings.Join(want, " ") {
		t.Errorf("remaining = %v, want %v", remaining, want)
	}
}

func TestRunDefaultsTheWorkingDirectoryToTheCurrentOne(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got := workingDirectory(); got != filepath.Clean(cwd) {
		t.Errorf("working directory = %q, want %q", got, filepath.Clean(cwd))
	}
}

func TestRunAcceptsALogLevel(t *testing.T) {
	code, err, stdout, _ := run(t, "npm", "--skip.npm", "--logLevel", "error",
		"--settings", filepath.Join(t.TempDir(), "absent.xml"))

	if code != exitOK || err != nil {
		t.Fatalf("code = %d, err = %v; want 0 and no error", code, err)
	}
	if strings.Contains(stdout, "Skipping execution.") {
		t.Errorf("an INFO record survived a raised threshold:\n%s", stdout)
	}
}
