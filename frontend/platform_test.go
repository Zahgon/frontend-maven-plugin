package frontend

import "testing"

const (
	nodeVersion8  = "v8.17.2"
	nodeVersion15 = "v15.14.0"
	nodeVersion16 = "v16.1.0"
)

func TestDetectWinDoesntLookForAlpine(t *testing.T) {
	// The original declares a supplier that throws if called and then, as
	// written, passes a plain false one; the input is carried over as it stands.
	platform := GuessPlatformWith(OSWindows, ArchX86, func() bool { return false })
	if got := platform.NodeClassifier(nodeVersion15); got != "win-x86" {
		t.Errorf("NodeClassifier = %q, want %q", got, "win-x86")
	}
}

func TestDetectArmMacDownloadX64BinaryNode15(t *testing.T) {
	platform := GuessPlatformWith(OSMac, ArchARM64, func() bool { return false })
	if got := platform.NodeClassifier(nodeVersion15); got != "darwin-x64" {
		t.Errorf("NodeClassifier = %q, want %q", got, "darwin-x64")
	}
}

func TestDetectArmMacDownloadX64BinaryNode16(t *testing.T) {
	platform := GuessPlatformWith(OSMac, ArchARM64, func() bool { return false })
	if got := platform.NodeClassifier(nodeVersion16); got != "darwin-arm64" {
		t.Errorf("NodeClassifier = %q, want %q", got, "darwin-arm64")
	}
}

func TestDetectLinuxNotAlpine(t *testing.T) {
	platform := GuessPlatformWith(OSLinux, ArchX86, func() bool { return false })
	if got := platform.NodeClassifier(nodeVersion15); got != "linux-x86" {
		t.Errorf("NodeClassifier = %q, want %q", got, "linux-x86")
	}
	if got := platform.NodeDownloadRoot(); got != "https://nodejs.org/dist/" {
		t.Errorf("NodeDownloadRoot = %q, want %q", got, "https://nodejs.org/dist/")
	}
}

func TestDetectLinuxAlpine(t *testing.T) {
	platform := GuessPlatformWith(OSLinux, ArchX86, func() bool { return true })
	if got := platform.NodeClassifier(nodeVersion15); got != "linux-x86-musl" {
		t.Errorf("NodeClassifier = %q, want %q", got, "linux-x86-musl")
	}
	want := "https://unofficial-builds.nodejs.org/download/release/"
	if got := platform.NodeDownloadRoot(); got != want {
		t.Errorf("NodeDownloadRoot = %q, want %q", got, want)
	}
}

func TestDetectAixPpc64(t *testing.T) {
	platform := GuessPlatformWith(OSAIX, ArchPPC64, func() bool { return false })
	if got := platform.NodeClassifier(nodeVersion15); got != "aix-ppc64" {
		t.Errorf("NodeClassifier = %q, want %q", got, "aix-ppc64")
	}
}

func TestGetNodeMajorVersion(t *testing.T) {
	for _, testCase := range []struct {
		version string
		want    int
	}{
		{nodeVersion8, 8},
		{nodeVersion15, 15},
		{nodeVersion16, 16},
	} {
		got, ok := NodeMajorVersion(testCase.version)
		if !ok || got != testCase.want {
			t.Errorf("NodeMajorVersion(%q) = %d, %t; want %d, true", testCase.version, got, ok, testCase.want)
		}
	}
}

func TestOperatingSystemsNameThemselves(t *testing.T) {
	for os, want := range map[OS]string{
		OSWindows: "Windows",
		OSMac:     "Mac",
		OSLinux:   "Linux",
		OSSunOS:   "SunOS",
		OSAIX:     "AIX",
	} {
		if got := os.String(); got != want {
			t.Errorf("OS.String() = %q, want %q", got, want)
		}
	}
	for architecture, want := range map[Architecture]string{
		ArchX86: "x86", ArchX64: "x64", ArchARM64: "arm64", ArchARMv7l: "armv7l",
		ArchPPC: "ppc", ArchPPC64: "ppc64", ArchPPC64LE: "ppc64le", ArchS390X: "s390x",
	} {
		if got := architecture.String(); got != want {
			t.Errorf("Architecture.String() = %q, want %q", got, want)
		}
	}
}

func TestGuessedPlatformMatchesTheRunningMachine(t *testing.T) {
	// The alpine probe answers from the filesystem; on a machine that is not
	// Alpine it must say so, and on one that is it must say that too.
	if got, want := checkForAlpine(), exists(alpineReleaseFile); got != want {
		t.Errorf("checkForAlpine = %t, want %t", got, want)
	}

	platform := GuessPlatform()
	if platform.IsWindows() != (GuessOS() == OSWindows) {
		t.Error("the guessed platform disagrees with the guessed operating system")
	}
	if platform.Codename() != GuessOS().Codename() {
		t.Errorf("codename = %q, want %q", platform.Codename(), GuessOS().Codename())
	}
}

func TestArchitectureIsGuessedFromTheJvmSpellings(t *testing.T) {
	for _, testCase := range []struct {
		arch, version string
		want          Architecture
	}{
		{"ppc64le", "", ArchPPC64LE},
		{"aarch64", "", ArchARM64},
		{"s390x", "", ArchS390X},
		{"arm", "5.15.0-v7l", ArchARMv7l},
		{"arm", "5.15.0", ArchARM64},
		{"ppc64", "", ArchPPC64},
		{"ppc", "", ArchPPC},
		{"amd64", "", ArchX64},
		{"x86_64", "", ArchX64},
		{"i386", "", ArchX86},
		{"x86", "", ArchX86},
	} {
		if got := guessArchitecture(testCase.arch, testCase.version); got != testCase.want {
			t.Errorf("guessArchitecture(%q, %q) = %v, want %v",
				testCase.arch, testCase.version, got, testCase.want)
		}
	}
	if got := javaOSArch(); got == "" {
		t.Error("the running architecture has no JVM spelling")
	}
	// Only the 32-bit ARM branch reads the kernel version, and only on Linux.
	_ = javaOSVersion()
}

func TestOperatingSystemIsGuessedFromTheJvmSpellings(t *testing.T) {
	for name, want := range map[string]OS{
		"Windows 11": OSWindows,
		"Mac OS X":   OSMac,
		"SunOS":      OSSunOS,
		"AIX":        OSAIX,
		"aix":        OSAIX,
		"Linux":      OSLinux,
		"FreeBSD":    OSLinux,
	} {
		if got := guessOS(name); got != want {
			t.Errorf("guessOS(%q) = %v, want %v", name, got, want)
		}
	}
	if got := javaOSName(); got == "" {
		t.Error("the running platform has no JVM spelling")
	}
}

func TestPlatformBuildsDownloadPathsForWindows(t *testing.T) {
	for architecture, want := range map[Architecture]string{
		ArchX64:   "v18.0.0/win-x64/node.exe",
		ArchARM64: "v18.0.0/win-arm64/node.exe",
		ArchX86:   "v18.0.0/win-x86/node.exe",
	} {
		platform := NewPlatform(OSWindows, architecture)
		if got := platform.NodeDownloadFilename("v18.0.0", false); got != want {
			t.Errorf("download filename = %q, want %q", got, want)
		}
		if got := platform.LongNodeFilename("v18.0.0", false); got != "node.exe" {
			t.Errorf("long filename = %q, want %q", got, "node.exe")
		}
	}
	platform := NewPlatform(OSWindows, ArchX64)
	if got := platform.NodeDownloadFilename("v18.0.0", true); got != "v18.0.0/node-v18.0.0-win-x64.zip" {
		t.Errorf("archive download filename = %q, want the zip", got)
	}
	if got := platform.ArchiveExtension(); got != "zip" {
		t.Errorf("archive extension = %q, want %q", got, "zip")
	}
}

func TestNodeMajorVersionRejectsAMalformedVersion(t *testing.T) {
	for _, version := range []string{"", "v", "vXX", "18.0.0", "v18"} {
		if _, ok := NodeMajorVersion(version); ok {
			t.Errorf("NodeMajorVersion(%q) reported a version, want none", version)
		}
	}
}
