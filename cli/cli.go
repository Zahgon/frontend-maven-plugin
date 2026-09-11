// Package cli turns command-line arguments into a configured goal and runs it.
//
// It is the piece the Maven container used to supply: parameter binding, the
// settings file, the project directories, and the lifecycle phase a goal
// believes it is running in.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/goal"
	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// Version is the plugin version, matched to the project it was ported from.
const Version = "2.0.3-SNAPSHOT"

// Exit codes: a build failure is 1, a usage error is 2.
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

// Run parses args, runs the named goal, and reports the process exit code.
func Run(args []string, stdout, stderr io.Writer) (int, error) {
	restore := logging.SetOutput(stdout)
	defer logging.SetOutput(restore)

	properties, remaining := extractSystemProperties(args)

	if len(remaining) == 0 {
		writeUsage(stdout)
		return exitUsage, nil
	}

	switch remaining[0] {
	case "-h", "--help", "help":
		writeUsage(stdout)
		return exitOK, nil
	case "-v", "--version", "--Version", "version":
		fmt.Fprintln(stdout, "frontend "+Version)
		return exitOK, nil
	}

	name := remaining[0]
	definition, found := goal.Lookup(name)
	if !found {
		writeUsage(stderr)
		return exitUsage, fmt.Errorf("unknown goal %q", name)
	}

	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	target := definition.New(flags)

	settingsPath := flags.String("settings", goal.DefaultSettingsPath(),
		"path to the settings file holding proxies and server credentials")
	localRepository := flags.String("localRepository", goal.DefaultLocalRepository(),
		"local artifact repository that downloads are cached in")
	lifecyclePhase := flags.String("lifecyclePhase", definition.DefaultPhase,
		"lifecycle phase this goal is running in; decides whether skipTests applies")
	logLevel := flags.String("logLevel", "info", "log level: debug, info, warn or error")

	if err := flags.Parse(remaining[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK, nil
		}
		return exitUsage, nil
	}
	if flags.NArg() > 0 {
		return exitUsage, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	if missing := requiredOf(definition, target); len(missing) > 0 {
		return exitUsage, fmt.Errorf("goal %s requires --%s", name, strings.Join(missing, " and --"))
	}

	logging.SetLevel(logging.ParseLevel(*logLevel))

	settings, err := goal.LoadSettings(*settingsPath)
	if err != nil {
		return exitFailure, fmt.Errorf("could not read %s: %w", *settingsPath, err)
	}

	base := target.Params()
	if base.WorkingDirectory == "" {
		base.WorkingDirectory = workingDirectory()
	}
	base.Session = &goal.Session{
		Settings:                    settings,
		BaseDir:                     base.WorkingDirectory,
		MultiModuleProjectDirectory: base.WorkingDirectory,
		ExecutionRootDirectory:      workingDirectory(),
		LocalRepository:             *localRepository,
		LifecyclePhase:              *lifecyclePhase,
		SystemProperties:            properties,
	}

	if err := goal.Execute(target); err != nil {
		return exitFailure, err
	}
	return exitOK, nil
}

// requiredOf lists a goal's unsatisfied mandatory parameters. Definition.Required
// is nil for the goals that have none, so the check goes through here.
func requiredOf(definition goal.Definition, target goal.Goal) []string {
	if definition.Required == nil {
		return nil
	}
	return definition.Required(target)
}

// extractSystemProperties pulls -Dname=value settings out of the argument list,
// leaving the rest for the flag parser. They stand in for the JVM system
// properties a goal can be overridden with at run time.
func extractSystemProperties(args []string) (map[string]string, []string) {
	properties := map[string]string{}
	remaining := make([]string, 0, len(args))
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-D") || arg == "-D" {
			remaining = append(remaining, arg)
			continue
		}
		key, value, found := strings.Cut(strings.TrimPrefix(arg, "-D"), "=")
		if !found {
			value = "true"
		}
		properties[key] = value
	}
	return properties, remaining
}

func workingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Clean(cwd)
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "frontend "+Version+" - install Node.js and run front-end tasks")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  frontend <goal> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Goals:")

	definitions := goal.Definitions()
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	width := 0
	for _, definition := range definitions {
		if len(definition.Name) > width {
			width = len(definition.Name)
		}
	}
	for _, definition := range definitions {
		fmt.Fprintf(w, "  %-*s  %s\n", width, definition.Name, definition.Description)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'frontend <goal> --help' for a goal's options.")
	fmt.Fprintln(w, "Options are named after the plugin's parameter properties, so -DnodeVersion=v18.0.0")
	fmt.Fprintln(w, "becomes --nodeVersion v18.0.0. Use -Dname=value to set a system property.")
}
