package main

import (
	"log"

	"github.com/dgrinbergs/universal-music-library/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
