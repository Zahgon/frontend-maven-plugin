package goal

// The eighteen goal names, exactly as the plugin descriptor spells them.
//
// Each name is used twice — by the goal's Name method and by the command-line
// definition that constructs it — so it is declared once here and the two can
// never drift apart.
const (
	nameInstallNodeAndNpm      = "install-node-and-npm"
	nameInstallNodeAndYarn     = "install-node-and-yarn"
	nameInstallNodeAndPnpm     = "install-node-and-pnpm"
	nameInstallNodeAndCorepack = "install-node-and-corepack"
	nameInstallBun             = "install-bun"

	nameNpm      = "npm"
	nameNpx      = "npx"
	namePnpm     = "pnpm"
	nameYarn     = "yarn"
	nameBun      = "bun"
	nameCorepack = "corepack"

	nameBower = "bower"
	nameJspm  = "jspm"
	nameKarma = "karma"

	nameGrunt   = "grunt"
	nameGulp    = "gulp"
	nameEmber   = "ember"
	nameWebpack = "webpack"
)
