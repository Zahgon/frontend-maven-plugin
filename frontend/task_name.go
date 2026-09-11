package frontend

import (
	"regexp"
	"strings"
)

// NodeTaskRunner is one front-end task this plugin can run.
type NodeTaskRunner interface {
	Execute(args string, environment map[string]string) error
}

const (
	doubleSlash = "//"
	atSign      = "@"
)

var taskNameFromLocation = regexp.MustCompile(`^.*/([^/]+)(?:\.js)?$`)

// taskNameFor derives a task's display name from where its script lives.
//
// The greedy prefix means the optional ".js" group always matches empty, so
// "node_modules/gulp/bin/gulp.js" names the task "gulp.js" rather than "gulp".
// That name reaches the log and the failure message, so the quirk is preserved.
func taskNameFor(taskLocation string) string {
	match := taskNameFromLocation.FindStringSubmatch(taskLocation)
	if match == nil {
		return taskLocation
	}
	return match[1]
}

// taskToString renders a task and its arguments for the log, masking any proxy
// password on the way through.
func taskToString(taskName string, arguments []string) string {
	clonedArguments := make([]string, len(arguments))
	copy(clonedArguments, arguments)
	for i, s := range clonedArguments {
		if strings.Contains(s, "proxy=") {
			clonedArguments[i] = maskPassword(s)
		}
	}
	return "'" + taskName + " " + Implode(" ", clonedArguments) + "'"
}

// maskPassword replaces the password inside a proxy URL with "***".
//
// It is a best-effort scan rather than a URL parse: the value has to carry a
// scheme, a "//" and an "@", with the "//" before the last "@", for anything to
// be masked at all.
func maskPassword(proxyString string) string {
	retVal := proxyString
	if strings.TrimSpace(proxyString) == "" {
		return retVal
	}
	hasSchemeDefined := strings.Contains(proxyString, "http:") || strings.Contains(proxyString, "https:")
	hasProtocolDefined := strings.Contains(proxyString, doubleSlash)
	hasAtCharacterDefined := strings.Contains(proxyString, atSign)
	if !(hasSchemeDefined && hasProtocolDefined && hasAtCharacterDefined) {
		return retVal
	}
	firstDoubleSlashIndex := strings.Index(proxyString, doubleSlash)
	lastAtCharIndex := strings.LastIndex(proxyString, atSign)
	if firstDoubleSlashIndex >= lastAtCharIndex {
		return retVal
	}
	startOfUserNameIndex := firstDoubleSlashIndex + len(doubleSlash)
	userInfo := proxyString[startOfUserNameIndex:lastAtCharIndex]
	userParts := strings.Split(userInfo, ":")
	if len(userParts) == 0 {
		return retVal
	}
	firstColonInUsernameOrEndOfUserNameIndex := startOfUserNameIndex + len(userParts[0])
	leftPart := proxyString[:firstColonInUsernameOrEndOfUserNameIndex]
	rightPart := proxyString[lastAtCharIndex:]
	return leftPart + ":***" + rightPart
}
