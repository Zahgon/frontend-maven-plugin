# frontend-maven-plugin

[![Build](https://github.com/eirslett/frontend-maven-plugin/actions/workflows/main.yml/badge.svg)](https://github.com/eirslett/frontend-maven-plugin/actions/workflows/main.yml)

This tool downloads/installs Node and NPM locally for your project, runs `npm install`, and then any combination of
[Bower](http://bower.io/), [Grunt](http://gruntjs.com/), [Gulp](http://gulpjs.com/), [Jspm](http://jspm.io),
[Karma](http://karma-runner.github.io/), or [Webpack](http://webpack.github.io/).
It's supposed to work on Windows, OS X and Linux.

If you prefer [Yarn](https://yarnpkg.com/) over [NPM](https://www.npmjs.com/) for your node package fetching,
this tool can also download Node and Yarn and then run `yarn install` for your project.

#### What is this meant to do?

- Let you keep your frontend and backend builds as separate as possible, by
  reducing the amount of interaction between them to the bare minimum; using only 1 tool.
- Let you use Node.js and its libraries in your build process without installing Node/NPM
  globally for your build system
- Let you ensure that the version of Node and NPM being run is the same in every build environment

#### What is this not meant to do?

- Not meant to replace the developer version of Node - frontend developers will still install Node on their
  laptops, but backend developers can run a clean build without even installing Node on their computer.
- Not meant to install Node for production uses. The Node usage is intended as part of a frontend build,
  running common javascript tasks such as minification, obfuscation, compression, packaging, testing etc.

**Notice:** _This tool does not support already installed Node or npm versions._

## Requirements

- _Go 1.23_ or newer to build it. Nothing else: the built binary has no runtime dependencies.

## Installation

```bash
go install github.com/eirslett/frontend-maven-plugin/cmd/frontend@latest
```

or, from a checkout:

```bash
go build -o dist/frontend ./cmd/frontend
```

## Usage

Every command is one **goal**, named exactly as it was when this project was a Maven plugin, and every option is
named after that goal's parameter property. A build that used to pass `-DnodeVersion=v18.0.0` now passes
`--nodeVersion v18.0.0`.

```bash
frontend --help                 # list the goals
frontend <goal> --help          # list a goal's options
frontend --version
```

Options shared by every goal:

| Option | Default | Meaning |
| --- | --- | --- |
| `--workingDirectory` | the current directory | Where `package.json` and your frontend config files live |
| `--installDirectory` | the working directory | Where node and the package managers are installed |
| `--environmentVariables NAME=VALUE` | *(none)* | Extra environment for the spawned tools; repeatable |
| `--skipTests` | `false` | Skip when running in a testing phase |
| `--maven.test.failure.ignore` | `false` | Downgrade a task failure in a testing phase to a logged error |
| `--lifecyclePhase` | `generate-resources`, or `test` for `karma` | The phase the goal believes it is running in |
| `--settings` | `~/.m2/settings.xml` | Settings file holding proxies and server credentials |
| `--localRepository` | `~/.m2/repository` | Where downloaded archives are cached between builds |
| `--logLevel` | `info` | `debug`, `info`, `warn` or `error` |
| `-Dname=value` | | Set a system property; `-DnpmRegistryURL=…` overrides the configured registry |

### Installing node and npm

```bash
frontend install-node-and-npm --nodeVersion v18.0.0 --npmVersion 8.6.0
```

| Option | Default |
| --- | --- |
| `--nodeVersion` | **required**; most Node version names start with `v` |
| `--npmVersion` | `provided` — use the npm bundled with that Node release |
| `--nodeDownloadRoot` | `https://nodejs.org/dist/` |
| `--npmDownloadRoot` | `https://registry.npmjs.org/npm/-/` |
| `--downloadRoot` | *deprecated*; used only when neither root above is given |
| `--serverId` | server entry holding the download username and password |
| `--skip.installnodenpm` | `false` |

On Alpine the download root switches automatically to
`https://unofficial-builds.nodejs.org/download/release/` and the classifier gains a `-musl` suffix.

### Installing node and yarn

```bash
frontend install-node-and-yarn --nodeVersion v18.0.0 --yarnVersion v1.22.19
```

| Option | Default |
| --- | --- |
| `--nodeVersion` | **required** |
| `--yarnVersion` | **required**; has to start with `v` |
| `--nodeDownloadRoot` | `https://nodejs.org/dist/` |
| `--yarnDownloadRoot` | `https://github.com/yarnpkg/yarn/releases/download/` |
| `--serverId` | server entry holding the download username and password |
| `--skip.installyarn` | `false` |

Yarn Berry is detected from a `.yarnrc.yml` at the project root, the multi-module root, the execution root, or the
working directory.

### Installing node and pnpm

```bash
frontend install-node-and-pnpm --nodeVersion v18.0.0 --pnpmVersion 7.0.0
```

| Option | Default |
| --- | --- |
| `--nodeVersion` | **required** |
| `--pnpmVersion` | **required**; `v1.2.3` and `1.2.3` are both valid |
| `--nodeDownloadRoot` | `https://nodejs.org/dist/` |
| `--pnpmDownloadRoot` | `https://registry.npmjs.org/pnpm/-/` |
| `--downloadRoot` | *deprecated* |
| `--serverId` | server entry holding the download username and password |
| `--skip.installnodepnpm` | `false` |

### Installing node and corepack

```bash
frontend install-node-and-corepack --nodeVersion v18.0.0
```

| Option | Default |
| --- | --- |
| `--nodeVersion` | **required** |
| `--corepackVersion` | `provided` — use the corepack bundled with that Node release |
| `--nodeDownloadRoot` | `https://nodejs.org/dist/` |
| `--corepackDownloadRoot` | `https://registry.npmjs.org/corepack/-/` |
| `--serverId` | server entry holding the download username and password |
| `--skip.installnodecorepack` | `false` |

### Installing bun

```bash
frontend install-bun --bunVersion v1.0.0
```

| Option | Default |
| --- | --- |
| `--bunVersion` | **required** |
| `--bunDownloadRoot` | `https://github.com/oven-sh/bun/releases/download/` |
| `--serverId` | server entry holding the download username and password |
| `--skip.installbun` | `false` |

### Running the package managers

```bash
frontend npm --frontend.npm.arguments "install"
frontend npx --frontend.npx.arguments "cowsay hello"
frontend pnpm --frontend.pnpm.arguments "install"
frontend yarn --frontend.yarn.arguments "install"
frontend bun --frontend.bun.arguments "install"
frontend corepack --frontend.corepack.arguments "enable"
```

| Goal | Arguments option | Default | Skip | Inherit proxies |
| --- | --- | --- | --- | --- |
| `npm` | `--frontend.npm.arguments` | `install` | `--skip.npm` | `--frontend.npm.npmInheritsProxyConfigFromMaven` |
| `npx` | `--frontend.npx.arguments` | `install` | `--skip.npx` | `--frontend.npx.npmInheritsProxyConfigFromMaven` |
| `pnpm` | `--frontend.pnpm.arguments` | `install` | `--skip.pnpm` | `--frontend.pnpm.pnpmInheritsProxyConfigFromMaven` |
| `yarn` | `--frontend.yarn.arguments` | *(empty)* | `--skip.yarn` | `--frontend.yarn.yarnInheritsProxyConfigFromMaven` |
| `bun` | `--frontend.bun.arguments` | *(empty)* | `--skip.bun` | `--frontend.bun.bunInheritsProxyConfigFromMaven` |
| `corepack` | `--frontend.corepack.arguments` | `enable` | `--skip.corepack` | *(none)* |

`npm`, `npx`, `pnpm`, `yarn` and `bun` also take `--npmRegistryURL`, which `-DnpmRegistryURL=…` overrides at run time.

### Running the build tools

```bash
frontend grunt   --frontend.grunt.arguments   "--no-color"
frontend gulp    --frontend.gulp.arguments    "build"
frontend ember   --frontend.ember.arguments   "build"
frontend webpack --frontend.webpack.arguments "--config webpack.config.js"
frontend bower   --frontend.bower.arguments   "install"
frontend jspm    --frontend.bower.arguments   "install"
frontend karma   --karmaConfPath              "karma.conf.js"
```

`grunt`, `gulp`, `ember` and `webpack` take `--srcdir`, `--outputdir` and a repeatable `--triggerfiles`. Under an
incremental build they run only when a trigger file changed, when no `srcdir` is set, or when the `srcdir` scan finds
anything. The default trigger file is `Gruntfile.js` for `grunt` and `ember`, `gulpfile.js` for `gulp`, and
`webpack.config.js` for `webpack`.

### Optional Configuration

#### Working directory

The working directory is where you've put `package.json` and your frontend configuration files (`Gruntfile.js` or
`gulpfile.js` etc). The default working directory is the directory you run the command from. You can change it:

```bash
frontend npm --workingDirectory src/main/frontend
```

**Notice:** _Npm packages will always be installed in `node_modules` next to your `package.json`, which is default npm behavior._

#### Installation Directory

The installation directory is the folder where your node and npm are installed. It defaults to the working directory:

```bash
frontend install-node-and-npm --nodeVersion v18.0.0 --installDirectory target
```

#### Proxy settings

If you have [configured proxy settings](http://maven.apache.org/guides/mini/guide-proxies.html) in your
`settings.xml`, the tool will automatically use the proxy for downloading node and npm, as well as
[passing the proxy to npm commands](https://docs.npmjs.com/misc/config#proxy).

**Non Proxy Hosts:** npm does not currently support non proxy hosts - if you are using a proxy and npm install
is not downloading from your repository, it may be because it cannot be accessed through your proxy.
If that is the case, you can stop the npm execution from inheriting the proxy settings:

```bash
frontend npm --frontend.npm.npmInheritsProxyConfigFromMaven=false
```

The same switch exists for bower (`--frontend.bower.bowerInheritsProxyConfigFromMaven=false`),
yarn (`--frontend.yarn.yarnInheritsProxyConfigFromMaven=false`), pnpm and bun.

Passwords encrypted with `settings-security.xml` are **not** decrypted; use a plaintext `settings.xml`
or an environment-provided credential.

#### Environment variables

If you need to pass some variable to Node, use `--environmentVariables`, once per variable:

```bash
frontend npm --environmentVariables Jon=Snow \
             --environmentVariables Tyrion=Lannister \
             --environmentVariables NODE_ENV=production
```

#### Ignoring Failure

**Ignoring failed tests:** to ignore test failures in a testing phase:

```bash
frontend karma --lifecyclePhase test --maven.test.failure.ignore
```

#### Skipping Execution

Each frontend build tool, package manager, and installation goal allows skipping execution.
This is useful for projects that contain multiple builds.

**Note** that if the installation goals or package manager (npm or yarn) are skipped, other build tools will also need
to be skipped because they would not have been downloaded.
For example, in a project using npm and gulp, if npm is skipped, gulp must also be skipped or the build will fail.

Installation goals and the flag to enable skipping:

- install-node-and-npm `--skip.installnodenpm`
- install-node-and-yarn `--skip.installyarn`
- install-node-and-pnpm `--skip.installnodepnpm`
- install-node-and-corepack `--skip.installnodecorepack`
- install-bun `--skip.installbun`

Build tools and the flag to enable skipping:

- npm `--skip.npm`
- yarn `--skip.yarn`
- bower `--skip.bower`
- bun `--skip.bun`
- corepack `--skip.corepack`
- ember `--skip.ember`
- grunt `--skip.grunt`
- gulp `--skip.gulp`
- jspm `--skip.jspm`
- karma `--skip.karma`
- npx `--skip.npx`
- pnpm `--skip.pnpm`
- webpack `--skip.webpack`

## Incremental builds

Goals support incremental builds to avoid doing unnecessary work. During an incremental build the `npm` goal will only
run if the `package.json` file has been changed. The `grunt` and `gulp` goals have `srcdir` and `triggerfiles` optional
configuration options; if these are set they check for changes in your source files before being run. A plain
command-line run is never incremental, so every goal executes.

## Project layout

| Directory | Contents |
| --- | --- |
| `cmd/frontend` | the command-line entry point |
| `cli` | argument parsing, settings loading, goal dispatch |
| `goal` | the eighteen goals and their parameters |
| `frontend` | the library: platform detection, downloads, extraction, installers, runners |
| `internal/logging` | the levelled logger the whole tool logs through |

## To build this project:

```bash
go build ./...
go test ./...
```

## Issues, Contributing

Please post any issues on the [Github's Issue tracker](https://github.com/eirslett/frontend-maven-plugin/issues).
You can find a full list of [contributors here](https://github.com/eirslett/frontend-maven-plugin/graphs/contributors).
The project is being maintained, but not actively. New features are only added if there is a popular demand for them and time allows. Development of this project has no financial backing, therefore it can sometimes take a while for changes to be merged and/or released.

Pull requests that fix security issues are generally welcome. Note that most security vulnerabilities in library dependencies are not that relevant to this tool, because it only processes trusted input (In the sense that we trust the official Node.js website to not contain malware.)

## License

[Apache 2.0](LICENSE)
