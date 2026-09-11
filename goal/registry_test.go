package goal

import (
	"flag"
	"io"
	"sort"
	"strings"
	"testing"
)

// The flag names are the original's parameter properties, so a build script's
// -D settings carry over unchanged. That mapping is asserted goal by goal.

func TestDefinitionsCoverEveryGoal(t *testing.T) {
	names := Names()
	want := []string{
		"bower", "bun", "corepack", "ember", "grunt", "gulp", "install-bun",
		"install-node-and-corepack", "install-node-and-npm", "install-node-and-pnpm",
		"install-node-and-yarn", "jspm", "karma", "npm", "npx", "pnpm", "webpack", "yarn",
	}
	sort.Strings(want)
	if len(names) != len(want) {
		t.Fatalf("goals = %v, want %d of them", names, len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("goal %d = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestLookupFindsAGoalByName(t *testing.T) {
	definition, found := Lookup("install-node-and-npm")
	if !found {
		t.Fatal("install-node-and-npm was not found")
	}
	if definition.DefaultPhase != GenerateResourcesPhase {
		t.Errorf("default phase = %q, want %q", definition.DefaultPhase, GenerateResourcesPhase)
	}
	if definition.Description == "" {
		t.Error("the goal has no description")
	}

	karma, found := Lookup("karma")
	if !found {
		t.Fatal("karma was not found")
	}
	if karma.DefaultPhase != TestPhase {
		t.Errorf("karma's default phase = %q, want %q", karma.DefaultPhase, TestPhase)
	}

	if _, found := Lookup("no-such-goal"); found {
		t.Error("an unknown goal must not be found")
	}
}

// flagNames lists the options a goal registers.
func flagNames(t *testing.T, name string) map[string]string {
	t.Helper()
	definition, found := Lookup(name)
	if !found {
		t.Fatalf("%s was not found", name)
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	definition.New(flags)
	registered := map[string]string{}
	flags.VisitAll(func(f *flag.Flag) { registered[f.Name] = f.DefValue })
	return registered
}

func TestEveryGoalRegistersTheSharedParameters(t *testing.T) {
	for _, name := range Names() {
		registered := flagNames(t, name)
		for _, shared := range []string{
			"skipTests", "maven.test.failure.ignore", "workingDirectory",
			"installDirectory", "environmentVariables",
		} {
			if _, ok := registered[shared]; !ok {
				t.Errorf("%s does not register --%s", name, shared)
			}
		}
	}
}

func TestGoalsRegisterTheirOwnParametersWithTheOriginalDefaults(t *testing.T) {
	for name, want := range map[string]map[string]string{
		"install-node-and-npm": {
			"nodeVersion": "", "npmVersion": "provided", "downloadRoot": "",
			"npmDownloadRoot":  "https://registry.npmjs.org/npm/-/",
			"nodeDownloadRoot": "", "serverId": "", "skip.installnodenpm": "false",
		},
		"install-node-and-yarn": {
			"nodeVersion": "", "yarnVersion": "",
			"yarnDownloadRoot": "https://github.com/yarnpkg/yarn/releases/download/",
			"skip.installyarn": "false",
		},
		"install-node-and-pnpm": {
			"pnpmVersion": "", "pnpmDownloadRoot": "https://registry.npmjs.org/pnpm/-/",
			"skip.installnodepnpm": "false",
		},
		"install-node-and-corepack": {
			"corepackVersion":          "provided",
			"corepackDownloadRoot":     "https://registry.npmjs.org/corepack/-/",
			"skip.installnodecorepack": "false",
		},
		"install-bun": {"bunVersion": "", "bunDownloadRoot": "", "skip.installbun": "false"},
		"npm": {
			"frontend.npm.arguments": "install", "npmRegistryURL": "", "skip.npm": "false",
			"frontend.npm.npmInheritsProxyConfigFromMaven": "true",
		},
		"npx": {
			"frontend.npx.arguments": "install", "skip.npx": "false",
			"frontend.npx.npmInheritsProxyConfigFromMaven": "true",
		},
		"pnpm": {
			"frontend.pnpm.arguments": "install", "skip.pnpm": "false",
			"frontend.pnpm.pnpmInheritsProxyConfigFromMaven": "true",
		},
		"yarn": {
			"frontend.yarn.arguments": "", "skip.yarn": "false",
			"frontend.yarn.yarnInheritsProxyConfigFromMaven": "true",
		},
		"bun": {
			"frontend.bun.arguments": "", "skip.bun": "false",
			"frontend.bun.bunInheritsProxyConfigFromMaven": "true",
		},
		"corepack": {"frontend.corepack.arguments": "enable", "skip.corepack": "false"},
		"bower": {
			"frontend.bower.arguments": "install", "skip.bower": "false",
			"frontend.bower.bowerInheritsProxyConfigFromMaven": "true",
		},
		"jspm":  {"frontend.bower.arguments": "install", "skip.jspm": "false"},
		"karma": {"karmaConfPath": "karma.conf.js", "skip.karma": "false"},
		"grunt": {
			"frontend.grunt.arguments": "", "triggerfiles": "", "srcdir": "", "outputdir": "",
			"skip.grunt": "false",
		},
		"gulp":    {"frontend.gulp.arguments": "", "skip.gulp": "false"},
		"ember":   {"frontend.ember.arguments": "", "skip.ember": "false"},
		"webpack": {"frontend.webpack.arguments": "", "skip.webpack": "false"},
	} {
		registered := flagNames(t, name)
		for option, defaultValue := range want {
			got, ok := registered[option]
			if !ok {
				t.Errorf("%s does not register --%s", name, option)
				continue
			}
			if got != defaultValue {
				t.Errorf("%s --%s defaults to %q, want %q", name, option, got, defaultValue)
			}
		}
	}
}

func TestRequiredParametersAreReported(t *testing.T) {
	for _, testCase := range []struct {
		name string
		want []string
	}{
		{"install-node-and-npm", []string{"nodeVersion"}},
		{"install-node-and-yarn", []string{"nodeVersion", "yarnVersion"}},
		{"install-node-and-pnpm", []string{"nodeVersion", "pnpmVersion"}},
		{"install-node-and-corepack", []string{"nodeVersion"}},
		{"install-bun", []string{"bunVersion"}},
	} {
		definition, _ := Lookup(testCase.name)
		flags := flag.NewFlagSet(testCase.name, flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		target := definition.New(flags)

		missing := definition.Required(target)

		if strings.Join(missing, ",") != strings.Join(testCase.want, ",") {
			t.Errorf("%s requires %v, want %v", testCase.name, missing, testCase.want)
		}
	}

	// A goal with no mandatory parameter declares none.
	npm, _ := Lookup("npm")
	if npm.Required != nil {
		t.Error("npm must declare no required parameters")
	}
}

func TestRequiredParametersAreSatisfiedByAValue(t *testing.T) {
	definition, _ := Lookup("install-node-and-yarn")
	flags := flag.NewFlagSet("install-node-and-yarn", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	target := definition.New(flags)
	requireNoError(t, flags.Parse([]string{"--nodeVersion", "v18.0.0", "--yarnVersion", "v1.22.19"}))

	if missing := definition.Required(target); len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
	if got := requiredWhenEmpty("name", "  "); len(got) != 1 {
		t.Errorf("a blank value must be reported as missing, got %v", got)
	}
}

func TestEnvironmentVariablesFlagCollectsPairs(t *testing.T) {
	definition, _ := Lookup("npm")
	flags := flag.NewFlagSet("npm", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	target := definition.New(flags)

	requireNoError(t, flags.Parse([]string{
		"--environmentVariables", "NODE_ENV=production",
		"--environmentVariables", "OTHER=value",
	}))

	variables := target.Params().EnvironmentVariables
	if variables["NODE_ENV"] != "production" || variables["OTHER"] != "value" {
		t.Errorf("environment = %v, want both pairs", variables)
	}
	if err := flags.Parse([]string{"--environmentVariables", "no-equals"}); err == nil {
		t.Error("a value without an equals sign must be rejected")
	}
}

func TestTriggerfilesFlagCollectsPaths(t *testing.T) {
	definition, _ := Lookup("grunt")
	flags := flag.NewFlagSet("grunt", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	target := definition.New(flags)

	requireNoError(t, flags.Parse([]string{"--triggerfiles", "a.js", "--triggerfiles", "b.js"}))

	grunt := target.(*Grunt)
	if len(grunt.Triggerfiles) != 2 || grunt.Triggerfiles[0] != "a.js" || grunt.Triggerfiles[1] != "b.js" {
		t.Errorf("trigger files = %v, want both paths", grunt.Triggerfiles)
	}
}

func TestRepeatableFlagValuesRenderThemselves(t *testing.T) {
	variables := map[string]string{}
	mapFlag := newMapValue(&variables)
	if got := mapFlag.String(); got != "" {
		t.Errorf("an empty map renders as %q, want empty", got)
	}
	requireNoError(t, mapFlag.Set("B=2"))
	requireNoError(t, mapFlag.Set("A=1"))
	if got := mapFlag.String(); got != "A=1,B=2" {
		t.Errorf("map renders as %q, want %q", got, "A=1,B=2")
	}
	var nilTarget *mapValue
	_ = nilTarget

	var list []string
	listFlag := newListValue(&list)
	if got := listFlag.String(); got != "" {
		t.Errorf("an empty list renders as %q, want empty", got)
	}
	requireNoError(t, listFlag.Set("a"))
	requireNoError(t, listFlag.Set("b"))
	if got := listFlag.String(); got != "a,b" {
		t.Errorf("list renders as %q, want %q", got, "a,b")
	}
}
