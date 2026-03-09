package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bssmnt/lazycron/internal/gui"
	"github.com/bssmnt/lazycron/internal/types"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("lazycron", types.Version)
		return
	}

	app, err := gui.New()
	if err != nil {
		log.Fatalf("failed to initialise lazycron: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("lazycron error: %v", err)
	}
}
