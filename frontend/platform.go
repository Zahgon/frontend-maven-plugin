package frontend

import (
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Architecture names the CPU architecture slice of a Node.js download
// classifier.
type Architecture string

// The architectures the download classifier can name.
const (
	ArchX86     Architecture = "x86"
	ArchX64     Architecture = "x64"
	ArchPPC64LE Architecture = "ppc64le"
	ArchS390X   Architecture = "s390x"
	ArchARM64   Architecture = "arm64"
	ArchARMv7l  Architecture = "armv7l"
	ArchPPC     Architecture = "ppc"
	ArchPPC64   Architecture = "ppc64"
)

// String is the name that appears in a download classifier.
func (a Architecture) String() string {
	return string(a)
}

// GuessArchitecture picks the architecture of the running machine.
//
// The decision is made on the JVM's os.arch and os.version spellings rather
// than on Go's own, because those spellings are what the original branched on
// and a divergence here changes the URL every download is fetched from.
func GuessArchitecture() Architecture {
	return guessArchitecture(javaOSArch(), javaOSVersion())
}

func guessArchitecture(arch, version string) Architecture {
	switch arch {
	case "ppc64le":
		return ArchPPC64LE
	case "aarch64":
		return ArchARM64
	case "s390x":
		return ArchS390X
	case "arm":
		if strings.Contains(version, "v7") {
			return ArchARMv7l
		}
		return ArchARM64
	case "ppc64":
		return ArchPPC64
	case "ppc":
		return ArchPPC
	default:
		if strings.Contains(arch, "64") {
			return ArchX64
		}
		return ArchX86
	}
}

// javaOSArchNames spell each Go architecture the way a JVM reports os.arch.
// Most spellings coincide; arm64 and 386 are the two that do not.
var javaOSArchNames = map[string]string{
	"amd64":   "amd64",
	"386":     "x86",
	"arm64":   "aarch64",
	"arm":     "arm",
	"ppc64":   "ppc64",
	"ppc64le": "ppc64le",
	"s390x":   "s390x",
}

// javaOSArch renders the running architecture the way a JVM reports os.arch, so
// that GuessArchitecture branches on the strings the original was written for.
// An architecture with no JVM spelling of its own keeps Go's.
func javaOSArch() string {
	if name, found := javaOSArchNames[runtime.GOARCH]; found {
		return name
	}
	return runtime.GOARCH
}

// javaOSVersion renders the kernel release the way a JVM reports os.version.
// Only the 32-bit ARM branch consults it, to tell an ARMv7 kernel from an ARMv8
// one, and only Linux publishes it in a file the standard library can read.
func javaOSVersion() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(release))
}

// OS names the operating-system slice of a Node.js download classifier.
type OS string

// The operating systems the download classifier can name.
const (
	OSWindows OS = "Windows"
	OSMac     OS = "Mac"
	OSLinux   OS = "Linux"
	OSSunOS   OS = "SunOS"
	OSAIX     OS = "AIX"
)

// String is the enum constant's name.
func (o OS) String() string {
	return string(o)
}

// GuessOS picks the operating system of the running machine.
func GuessOS() OS {
	return guessOS(javaOSName())
}

func guessOS(osName string) OS {
	switch {
	case strings.Contains(osName, "Windows"):
		return OSWindows
	case strings.Contains(osName, "Mac"):
		return OSMac
	case strings.Contains(osName, "SunOS"):
		return OSSunOS
	case strings.Contains(strings.ToUpper(osName), "AIX"):
		return OSAIX
	default:
		return OSLinux
	}
}

// javaOSNames spell each Go platform the way a JVM reports os.name. Anything
// not listed is a Unix that the original treats as Linux.
var javaOSNames = map[string]string{
	"windows": "Windows",
	"darwin":  "Mac OS X",
	"solaris": "SunOS",
	"illumos": "SunOS",
	"aix":     "AIX",
	"linux":   "Linux",
}

// javaOSName renders the running platform the way a JVM reports os.name.
func javaOSName() string {
	if name, found := javaOSNames[runtime.GOOS]; found {
		return name
	}
	return javaOSNames["linux"]
}

// ArchiveExtension is the extension of the Node.js distribution archive for
// this operating system.
func (o OS) ArchiveExtension() string {
	if o == OSWindows {
		return "zip"
	}
	return "tar.gz"
}

// codenames name the operating-system slice of a Node.js download classifier.
// Every download URL is built from one of these, so they are spelled once.
var codenames = map[OS]string{
	OSMac:     "darwin",
	OSWindows: "win",
	OSSunOS:   "sunos",
	OSAIX:     "aix",
	OSLinux:   "linux",
}

// Codename is the operating-system slice of a Node.js download classifier.
// Anything outside the five known systems is treated as Linux.
func (o OS) Codename() string {
	if name, found := codenames[o]; found {
		return name
	}
	return codenames[OSLinux]
}

// WindowsNodeExecutable is the file name of the Windows Node.js binary, which is
// downloaded bare unless an archive is asked for. It is also what the installer
// writes next to the install directory.
const WindowsNodeExecutable = "node.exe"

// nodeArchivePrefix opens the name of an unpacked Node.js distribution
// directory: "node-<version>-<classifier>".
const nodeArchivePrefix = "node-"

// nodeVersionThresholdMacARM64 is the first Node.js major version with an Apple
// silicon build; below it an arm64 Mac downloads the x64 distribution and runs
// it under Rosetta.
//
// https://github.com/nodejs/node/blob/master/doc/changelogs/CHANGELOG_V16.md#toolchain-and-compiler-upgrades
const nodeVersionThresholdMacARM64 = 16

const defaultNodeDownloadRoot = "https://nodejs.org/dist/"

// muslNodeDownloadRoot serves the unofficial musl builds. musl support is still
// experimental upstream, so the root stays overridable from configuration in
// case it moves before this project catches up.
//
// https://github.com/nodejs/node/blob/master/BUILDING.md#platform-list
const muslNodeDownloadRoot = "https://unofficial-builds.nodejs.org/download/release/"

const alpineReleaseFile = "/etc/alpine-release"

// Platform is the machine a Node.js distribution is being selected for: which
// build to download, from where, and what the unpacked directory will be called.
type Platform struct {
	nodeDownloadRoot string
	os               OS
	architecture     Architecture
	classifier       string
}

// NewPlatform describes a platform served by the official distribution.
func NewPlatform(os OS, architecture Architecture) *Platform {
	return NewPlatformWithRoot(defaultNodeDownloadRoot, os, architecture, "")
}

// NewPlatformWithRoot describes a platform completely, including the libc
// classifier that a non-glibc Linux needs in its download path.
func NewPlatformWithRoot(nodeDownloadRoot string, os OS, architecture Architecture, classifier string) *Platform {
	return &Platform{
		nodeDownloadRoot: nodeDownloadRoot,
		os:               os,
		architecture:     architecture,
		classifier:       classifier,
	}
}

// GuessPlatform describes the machine this process is running on.
func GuessPlatform() *Platform {
	return GuessPlatformWith(GuessOS(), GuessArchitecture(), checkForAlpine)
}

// checkForAlpine is the default musl probe: Alpine ships a release file no
// other distribution has, and it is the simplest check available.
func checkForAlpine() bool {
	return exists(alpineReleaseFile)
}

// GuessPlatformWith describes a platform, deciding on the libc from the supplied
// probe.
//
// The default libc is glibc, but Alpine uses musl. Where it is not the default,
// the Node.js download — and the path inside it — needs a classifier suffix such
// as "-musl", and the download root moves to the unofficial builds.
func GuessPlatformWith(os OS, architecture Architecture, checkForAlpine func() bool) *Platform {
	if os == OSLinux && checkForAlpine() {
		return NewPlatformWithRoot(muslNodeDownloadRoot, os, architecture, "musl")
	}
	return NewPlatform(os, architecture)
}

// NodeDownloadRoot is the base URL Node.js distributions are fetched from.
func (p *Platform) NodeDownloadRoot() string {
	return p.nodeDownloadRoot
}

// ArchiveExtension is the extension of this platform's distribution archive.
func (p *Platform) ArchiveExtension() string {
	return p.os.ArchiveExtension()
}

// Codename is the operating-system slice of the download classifier.
func (p *Platform) Codename() string {
	return p.os.Codename()
}

// IsWindows reports whether paths and executables take their Windows form.
func (p *Platform) IsWindows() bool {
	return p.os == OSWindows
}

// IsMac reports whether this is macOS.
func (p *Platform) IsMac() bool {
	return p.os == OSMac
}

// LongNodeFilename is the name of the directory the distribution unpacks into,
// or "node.exe" on Windows when the bare binary is downloaded instead of an
// archive.
func (p *Platform) LongNodeFilename(nodeVersion string, archiveOnWindows bool) string {
	if p.IsWindows() && !archiveOnWindows {
		return WindowsNodeExecutable
	}
	return nodeArchivePrefix + nodeVersion + "-" + p.NodeClassifier(nodeVersion)
}

// NodeDownloadFilename is the path, relative to the download root, of this
// platform's Node.js distribution.
func (p *Platform) NodeDownloadFilename(nodeVersion string, archiveOnWindows bool) string {
	if p.IsWindows() && !archiveOnWindows {
		switch p.architecture {
		case ArchX64:
			return nodeVersion + "/win-x64/" + WindowsNodeExecutable
		case ArchARM64:
			return nodeVersion + "/win-arm64/" + WindowsNodeExecutable
		default:
			return nodeVersion + "/win-x86/" + WindowsNodeExecutable
		}
	}
	return nodeVersion + "/" + p.LongNodeFilename(nodeVersion, archiveOnWindows) + "." + p.os.ArchiveExtension()
}

// NodeClassifier is the "<os>-<arch>[-<libc>]" slice that identifies a build.
func (p *Platform) NodeClassifier(nodeVersion string) string {
	result := p.Codename() + "-" + p.resolveArchitecture(nodeVersion).String()
	if p.classifier != "" {
		return result + "-" + p.classifier
	}
	return result
}

func (p *Platform) resolveArchitecture(nodeVersion string) Architecture {
	if p.IsMac() && p.architecture == ArchARM64 {
		major, ok := NodeMajorVersion(nodeVersion)
		if !ok || major < nodeVersionThresholdMacARM64 {
			return ArchX64
		}
	}
	return p.architecture
}

var nodeVersionPattern = regexp.MustCompile(`^v(\d+)\..*$`)

// NodeMajorVersion extracts the major version from a "vN.N.N" string. A version
// string in any other shape is malformed and reports false, which is the
// absent-value the original returns as a null Integer.
func NodeMajorVersion(nodeVersion string) (int, bool) {
	match := nodeVersionPattern.FindStringSubmatch(nodeVersion)
	if match == nil {
		return 0, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return major, true
}
