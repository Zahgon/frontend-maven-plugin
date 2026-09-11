package goal

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

// Definition is one goal as the command line sees it: its name, the lifecycle
// phase it binds to by default, a constructor, and the flags that carry its
// parameters.
//
// Flag names are the original's parameter property names verbatim, so that a
// build script's `-DnodeVersion=v18.0.0` becomes `--nodeVersion v18.0.0` with no
// other translation.
type Definition struct {
	// Name is the goal name.
	Name string
	// DefaultPhase is the lifecycle phase this goal binds to by default.
	DefaultPhase string
	// Description is the one-line summary the goal list shows.
	Description string
	// New builds the goal and registers its parameters on a flag set.
	New func(flags *flag.FlagSet) Goal
	// Required lists the parameters that have no default and must be supplied.
	Required func(g Goal) []string
}

// GenerateResourcesPhase is the phase seventeen of the eighteen goals bind to.
const GenerateResourcesPhase = "generate-resources"

// TestPhase is the phase the karma goal binds to.
const TestPhase = "test"

// bindBase registers the parameters every goal accepts.
func bindBase(flags *flag.FlagSet, base *Base) {
	flags.BoolVar(&base.SkipTests, "skipTests", false,
		"skip while running in the test phase")
	flags.BoolVar(&base.TestFailureIgnore, "maven.test.failure.ignore", false,
		"ignore a failure during testing; NOT RECOMMENDED, but quite convenient on occasion")
	flags.StringVar(&base.WorkingDirectory, "workingDirectory", "",
		"base directory for running all Node commands (usually the directory that contains package.json)")
	flags.StringVar(&base.InstallDirectory, "installDirectory", "",
		"base directory for installing node and npm")
	flags.Var(newMapValue(&base.EnvironmentVariables), "environmentVariables",
		"additional environment variable to pass to the build, as NAME=VALUE; repeatable")
}

// Definitions lists every goal, in the order the goal list prints them.
func Definitions() []Definition {
	return []Definition{
		{
			Name:         nameInstallNodeAndNpm,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Install Node.js and npm into the install directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &InstallNodeAndNpm{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.NodeDownloadRoot, "nodeDownloadRoot", "",
					"where to download the Node.js binary from; defaults to https://nodejs.org/dist/")
				flags.StringVar(&g.NpmDownloadRoot, "npmDownloadRoot", frontend.DefaultNpmDownloadRoot,
					"where to download the NPM binary from")
				flags.StringVar(&g.DownloadRoot, "downloadRoot", "",
					"deprecated: where to download Node.js and NPM binaries from")
				flags.StringVar(&g.NodeVersion, "nodeVersion", "",
					"version of Node.js to install; most names start with 'v', for example 'v0.10.18'")
				flags.StringVar(&g.NpmVersion, "npmVersion", "provided",
					"version of NPM to install")
				flags.StringVar(&g.ServerID, "serverId", "",
					"server id for the download username and password")
				flags.BoolVar(&g.Skip, "skip.installnodenpm", false, "skip execution of this goal")
				return g
			},
			Required: func(g Goal) []string {
				return requiredWhenEmpty("nodeVersion", g.(*InstallNodeAndNpm).NodeVersion)
			},
		},
		{
			Name:         nameInstallNodeAndYarn,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Install Node.js and Yarn into the install directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &InstallNodeAndYarn{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.NodeDownloadRoot, "nodeDownloadRoot", "",
					"where to download the Node.js binary from; defaults to https://nodejs.org/dist/")
				flags.StringVar(&g.YarnDownloadRoot, "yarnDownloadRoot", frontend.DefaultYarnDownloadRoot,
					"where to download the Yarn binary from")
				flags.StringVar(&g.NodeVersion, "nodeVersion", "",
					"version of Node.js to install; most names start with 'v', for example 'v0.10.18'")
				flags.StringVar(&g.YarnVersion, "yarnVersion", "",
					"version of Yarn to install; most names start with 'v', for example 'v0.15.0'")
				flags.StringVar(&g.ServerID, "serverId", "",
					"server id for the download username and password")
				flags.BoolVar(&g.Skip, "skip.installyarn", false, "skip execution of this goal")
				return g
			},
			Required: func(g Goal) []string {
				goal := g.(*InstallNodeAndYarn)
				return append(
					requiredWhenEmpty("nodeVersion", goal.NodeVersion),
					requiredWhenEmpty("yarnVersion", goal.YarnVersion)...)
			},
		},
		{
			Name:         nameInstallNodeAndPnpm,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Install Node.js and pnpm into the install directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &InstallNodeAndPnpm{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.NodeDownloadRoot, "nodeDownloadRoot", "",
					"where to download the Node.js binary from; defaults to https://nodejs.org/dist/")
				flags.StringVar(&g.PnpmDownloadRoot, "pnpmDownloadRoot", frontend.DefaultPnpmDownloadRoot,
					"where to download the pnpm binary from")
				flags.StringVar(&g.DownloadRoot, "downloadRoot", "",
					"deprecated: where to download Node.js and pnpm binaries from")
				flags.StringVar(&g.NodeVersion, "nodeVersion", "",
					"version of Node.js to install; most names start with 'v', for example 'v0.10.18'")
				flags.StringVar(&g.PnpmVersion, "pnpmVersion", "",
					"version of pnpm to install, with or without a leading 'v'")
				flags.StringVar(&g.ServerID, "serverId", "",
					"server id for the download username and password")
				flags.BoolVar(&g.Skip, "skip.installnodepnpm", false, "skip execution of this goal")
				return g
			},
			Required: func(g Goal) []string {
				goal := g.(*InstallNodeAndPnpm)
				return append(
					requiredWhenEmpty("nodeVersion", goal.NodeVersion),
					requiredWhenEmpty("pnpmVersion", goal.PnpmVersion)...)
			},
		},
		{
			Name:         nameInstallNodeAndCorepack,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Install Node.js and corepack into the install directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &InstallNodeAndCorepack{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.NodeDownloadRoot, "nodeDownloadRoot", "",
					"where to download the Node.js binary from; defaults to https://nodejs.org/dist/")
				flags.StringVar(&g.CorepackDownloadRoot, "corepackDownloadRoot", frontend.DefaultCorepackDownloadRoot,
					"where to download the corepack binary from")
				flags.StringVar(&g.NodeVersion, "nodeVersion", "",
					"version of Node.js to install; most names start with 'v', for example 'v0.10.18'")
				flags.StringVar(&g.CorepackVersion, "corepackVersion", "provided",
					"version of corepack to install, with or without a leading 'v'")
				flags.StringVar(&g.ServerID, "serverId", "",
					"server id for the download username and password")
				flags.BoolVar(&g.Skip, "skip.installnodecorepack", false, "skip execution of this goal")
				return g
			},
			Required: func(g Goal) []string {
				return requiredWhenEmpty("nodeVersion", g.(*InstallNodeAndCorepack).NodeVersion)
			},
		},
		{
			Name:         nameInstallBun,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Install Bun into the install directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &InstallBun{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.BunDownloadRoot, "bunDownloadRoot", "",
					"where to download the Bun binary from; defaults to https://github.com/oven-sh/bun/releases/download/")
				flags.StringVar(&g.BunVersion, "bunVersion", "",
					"version of Bun to install; most names start with 'v', for example 'v1.0.0'")
				flags.StringVar(&g.ServerID, "serverId", "",
					"server id for the download username and password")
				flags.BoolVar(&g.Skip, "skip.installbun", false, "skip execution of this goal")
				return g
			},
			Required: func(g Goal) []string {
				return requiredWhenEmpty("bunVersion", g.(*InstallBun).BunVersion)
			},
		},
		{
			Name:         nameNpm,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run npm in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Npm{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.npm.arguments", "install", "npm arguments")
				flags.BoolVar(&g.NpmInheritsProxyConfigFromMaven,
					"frontend.npm.npmInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to npm")
				flags.StringVar(&g.NpmRegistryURL, "npmRegistryURL", "",
					"registry override, passed as the registry option during npm install if set")
				flags.BoolVar(&g.Skip, "skip.npm", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameNpx,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run npx in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Npx{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.npx.arguments", "install", "npx arguments")
				flags.BoolVar(&g.NpmInheritsProxyConfigFromMaven,
					"frontend.npx.npmInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to npx")
				flags.StringVar(&g.NpmRegistryURL, "npmRegistryURL", "",
					"registry override, passed as the registry option during npm install if set")
				flags.BoolVar(&g.Skip, "skip.npx", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         namePnpm,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run pnpm in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Pnpm{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.pnpm.arguments", "install", "pnpm arguments")
				flags.BoolVar(&g.PnpmInheritsProxyConfigFromMaven,
					"frontend.pnpm.pnpmInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to pnpm")
				flags.StringVar(&g.PnpmRegistryURL, "npmRegistryURL", "",
					"registry override, passed as the registry option during pnpm install if set")
				flags.BoolVar(&g.Skip, "skip.pnpm", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameYarn,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run yarn in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Yarn{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.yarn.arguments", "", "yarn arguments")
				flags.BoolVar(&g.YarnInheritsProxyConfigFromMaven,
					"frontend.yarn.yarnInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to yarn")
				flags.StringVar(&g.NpmRegistryURL, "npmRegistryURL", "",
					"registry override, passed as the registry option during npm install if set")
				flags.BoolVar(&g.Skip, "skip.yarn", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameBun,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run bun in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Bun{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.bun.arguments", "", "bun arguments")
				flags.BoolVar(&g.BunInheritsProxyConfigFromMaven,
					"frontend.bun.bunInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to bun")
				flags.StringVar(&g.NpmRegistryURL, "npmRegistryURL", "",
					"registry override, passed as the registry option during npm install if set")
				flags.BoolVar(&g.Skip, "skip.bun", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameCorepack,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run corepack in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Corepack{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.corepack.arguments", "enable", "corepack arguments")
				flags.BoolVar(&g.Skip, "skip.corepack", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameBower,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run bower in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Bower{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.bower.arguments", "install", "bower arguments")
				flags.BoolVar(&g.BowerInheritsProxyConfigFromMaven,
					"frontend.bower.bowerInheritsProxyConfigFromMaven", true,
					"pass the build's proxy configuration on to bower")
				flags.BoolVar(&g.Skip, "skip.bower", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameJspm,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run jspm in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Jspm{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.Arguments, "frontend.bower.arguments", "install", "JSPM arguments")
				flags.BoolVar(&g.Skip, "skip.jspm", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameKarma,
			DefaultPhase: TestPhase,
			Description:  "Run karma in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Karma{}
				bindBase(flags, &g.Base)
				flags.StringVar(&g.KarmaConfPath, "karmaConfPath", "karma.conf.js",
					"path to the karma configuration file, relative to the working directory")
				flags.BoolVar(&g.Skip, "skip.karma", false, "skip execution of this goal")
				return g
			},
		},
		{
			Name:         nameGrunt,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run grunt in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Grunt{}
				bindBase(flags, &g.Base)
				bindWatched(flags, &g.watched, "frontend.grunt.arguments", "Grunt arguments", "skip.grunt")
				return g
			},
		},
		{
			Name:         nameGulp,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run gulp in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Gulp{}
				bindBase(flags, &g.Base)
				bindWatched(flags, &g.watched, "frontend.gulp.arguments", "Gulp arguments", "skip.gulp")
				return g
			},
		},
		{
			Name:         nameEmber,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run ember in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Ember{}
				bindBase(flags, &g.Base)
				bindWatched(flags, &g.watched, "frontend.ember.arguments", "Ember arguments", "skip.ember")
				return g
			},
		},
		{
			Name:         nameWebpack,
			DefaultPhase: GenerateResourcesPhase,
			Description:  "Run webpack in the working directory",
			New: func(flags *flag.FlagSet) Goal {
				g := &Webpack{}
				bindBase(flags, &g.Base)
				bindWatched(flags, &g.watched, "frontend.webpack.arguments", "Webpack arguments", "skip.webpack")
				return g
			},
		},
	}
}

func bindWatched(flags *flag.FlagSet, w *watched, argumentsProperty, argumentsUsage, skipProperty string) {
	flags.StringVar(&w.Arguments, argumentsProperty, "", argumentsUsage)
	flags.Var(newListValue(&w.Triggerfiles), "triggerfiles",
		"file to check for changes in addition to the srcdir files; repeatable")
	flags.StringVar(&w.Srcdir, "srcdir", "",
		"directory containing front-end files that will be processed")
	flags.StringVar(&w.Outputdir, "outputdir", "",
		"directory where front-end files will be written")
	flags.BoolVar(&w.Skip, skipProperty, false, "skip execution of this goal")
}

// Lookup finds a goal definition by name.
func Lookup(name string) (Definition, bool) {
	for _, definition := range Definitions() {
		if definition.Name == name {
			return definition, true
		}
	}
	return Definition{}, false
}

// Names lists every goal name, sorted.
func Names() []string {
	definitions := Definitions()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}
	sort.Strings(names)
	return names
}

func requiredWhenEmpty(name, value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{name}
	}
	return nil
}

// mapValue collects repeated NAME=VALUE flags into a map.
type mapValue struct {
	target *map[string]string
}

func newMapValue(target *map[string]string) *mapValue {
	return &mapValue{target: target}
}

func (v *mapValue) String() string {
	if v.target == nil || *v.target == nil {
		return ""
	}
	pairs := make([]string, 0, len(*v.target))
	for key, value := range *v.target {
		pairs = append(pairs, key+"="+value)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

func (v *mapValue) Set(raw string) error {
	key, value, found := strings.Cut(raw, "=")
	if !found {
		return fmt.Errorf("expected NAME=VALUE, got %q", raw)
	}
	if *v.target == nil {
		*v.target = map[string]string{}
	}
	(*v.target)[key] = value
	return nil
}

// listValue collects a repeated flag into a slice.
type listValue struct {
	target *[]string
}

func newListValue(target *[]string) *listValue {
	return &listValue{target: target}
}

func (v *listValue) String() string {
	if v.target == nil {
		return ""
	}
	return strings.Join(*v.target, ",")
}

func (v *listValue) Set(raw string) error {
	*v.target = append(*v.target, raw)
	return nil
}
