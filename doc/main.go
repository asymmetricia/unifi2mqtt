package main

import (
	"log"
	"os"

	"github.com/pdbogen/unifi2mqtt/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	_ = os.Mkdir("./docs", os.FileMode(0755))
	err := doc.GenMarkdownTree(cmd.Root, "./docs/")
	if err != nil {
		log.Fatal(err)
	}
}
