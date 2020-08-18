package main

import "github.com/pdbogen/unifi2mqtt/cmd"

var version = "dirty"

func main() {
	cmd.Root.Version = version
	_ = cmd.Root.Execute()
}
