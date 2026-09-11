package goal

import (
	"path/filepath"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

// The task goals. Four of them watch a source directory and a set of trigger
// files and stand down when an incremental build reports nothing changed; the
// other three simply run their tool.

// watched holds the parameters the four file-watching goals share.
type watched struct {
	// Arguments are the tool's arguments. Default is empty, which runs just the
	// bare command.
	Arguments string
	// Triggerfiles are files that should be checked for changes, in addition to
	// the srcdir files.
	Triggerfiles []string
	// Srcdir is the directory containing front-end files the tool will process.
	// If set, files in it are checked for modifications before the tool runs.
	Srcdir string
	// Outputdir is the directory the tool writes to. If set, it is refreshed
	// afterwards so the files show as modified in the IDE.
	Outputdir string
	// Skip skips execution of this goal.
	Skip bool
}

// run applies the shared watch-then-run-then-refresh sequence.
func (b *Base) runWatched(w *watched, tool, defaultTriggerfile string, execute func() error) error {
	triggerfiles := w.Triggerfiles
	if len(triggerfiles) == 0 {
		triggerfiles = []string{filepath.Join(b.WorkingDirectory, defaultTriggerfile)}
		w.Triggerfiles = triggerfiles
	}

	if !ShouldExecute(b.buildContext(), triggerfiles, w.Srcdir) {
		log.Info("Skipping " + tool + " as no modified files in " + w.Srcdir)
		return nil
	}

	if err := execute(); err != nil {
		return err
	}

	if w.Outputdir != "" {
		log.Info("Refreshing files after " + tool + ": " + w.Outputdir)
		b.buildContext().Refresh(w.Outputdir)
	}
	return nil
}

// Grunt is the grunt goal.
type Grunt struct {
	Base
	watched
}

// Name is the goal name.
func (g *Grunt) Name() string { return nameGrunt }

// Params exposes the shared parameters.
func (g *Grunt) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.grunt is set.
func (g *Grunt) SkipExecution() bool { return g.Skip }

// Run runs grunt when the watched files changed.
func (g *Grunt) Run(factory *frontend.FrontendPluginFactory) error {
	return g.runWatched(&g.watched, "grunt", "Gruntfile.js", func() error {
		return factory.GruntRunner().Execute(g.Arguments, g.EnvironmentVariables)
	})
}

// Gulp is the gulp goal.
type Gulp struct {
	Base
	watched
}

// Name is the goal name.
func (g *Gulp) Name() string { return nameGulp }

// Params exposes the shared parameters.
func (g *Gulp) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.gulp is set.
func (g *Gulp) SkipExecution() bool { return g.Skip }

// Run runs gulp when the watched files changed.
func (g *Gulp) Run(factory *frontend.FrontendPluginFactory) error {
	return g.runWatched(&g.watched, "gulp", "gulpfile.js", func() error {
		return factory.GulpRunner().Execute(g.Arguments, g.EnvironmentVariables)
	})
}

// Ember is the ember goal.
type Ember struct {
	Base
	watched
}

// Name is the goal name.
func (g *Ember) Name() string { return nameEmber }

// Params exposes the shared parameters.
func (g *Ember) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.ember is set.
func (g *Ember) SkipExecution() bool { return g.Skip }

// Run runs ember when the watched files changed. Its default trigger file is
// Gruntfile.js, which is what the original watches.
func (g *Ember) Run(factory *frontend.FrontendPluginFactory) error {
	return g.runWatched(&g.watched, "ember", "Gruntfile.js", func() error {
		return factory.EmberRunner().Execute(g.Arguments, g.EnvironmentVariables)
	})
}

// Webpack is the webpack goal.
type Webpack struct {
	Base
	watched
}

// Name is the goal name.
func (g *Webpack) Name() string { return nameWebpack }

// Params exposes the shared parameters.
func (g *Webpack) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.webpack is set.
func (g *Webpack) SkipExecution() bool { return g.Skip }

// Run runs webpack when the watched files changed.
func (g *Webpack) Run(factory *frontend.FrontendPluginFactory) error {
	return g.runWatched(&g.watched, "webpack", "webpack.config.js", func() error {
		return factory.WebpackRunner().Execute(g.Arguments, g.EnvironmentVariables)
	})
}

// Bower is the bower goal.
type Bower struct {
	Base
	// Arguments are the bower arguments. Default is "install".
	Arguments string
	// BowerInheritsProxyConfigFromMaven passes the build's proxies on to bower.
	BowerInheritsProxyConfigFromMaven bool
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Bower) Name() string { return nameBower }

// Params exposes the shared parameters.
func (g *Bower) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.bower is set.
func (g *Bower) SkipExecution() bool { return g.Skip }

// Run runs bower with the configured arguments.
func (g *Bower) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := g.proxyConfigFor(g.BowerInheritsProxyConfigFromMaven, "bower")
	return factory.BowerRunner(proxyConfig).Execute(g.Arguments, g.EnvironmentVariables)
}

// Jspm is the jspm goal.
type Jspm struct {
	Base
	// Arguments are the JSPM arguments. Default is "install".
	Arguments string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Jspm) Name() string { return nameJspm }

// Params exposes the shared parameters.
func (g *Jspm) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.jspm is set.
func (g *Jspm) SkipExecution() bool { return g.Skip }

// Run runs jspm with the configured arguments.
func (g *Jspm) Run(factory *frontend.FrontendPluginFactory) error {
	return factory.JspmRunner().Execute(g.Arguments, g.EnvironmentVariables)
}

// Karma is the karma goal. Unlike every other goal it binds to the test phase.
type Karma struct {
	Base
	// KarmaConfPath is the path to the karma configuration file, relative to
	// the working directory. Default is "karma.conf.js".
	KarmaConfPath string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Karma) Name() string { return nameKarma }

// Params exposes the shared parameters.
func (g *Karma) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.karma is set.
func (g *Karma) SkipExecution() bool { return g.Skip }

// Run starts karma against the configured configuration file.
func (g *Karma) Run(factory *frontend.FrontendPluginFactory) error {
	return factory.KarmaRunner().Execute("start "+g.KarmaConfPath, g.EnvironmentVariables)
}
