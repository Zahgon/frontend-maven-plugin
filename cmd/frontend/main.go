// Command frontend installs Node.js and a JavaScript package manager into a
// project and runs front-end tasks against that private installation.
//
// Each sub-command is one of the goals the original Maven plugin exposes, and
// every option is named after that goal's parameter property, so a build script
// that passed -DnodeVersion=v18.0.0 to Maven now passes --nodeVersion v18.0.0
// here.
package main

import (
	"fmt"
	"os"

	"github.com/eirslett/frontend-maven-plugin/cli"
)

func main() {
	code, err := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] "+err.Error())
	}
	os.Exit(code)
}
