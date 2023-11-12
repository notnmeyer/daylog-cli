package main

import "github.com/notnmeyer/daylog-cli/cmd"

var version, commit string

func main() {
	cmd.Execute(version, commit)
}
