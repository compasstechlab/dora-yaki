package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	app "github.com/compasstechlab/dora-yaki"
)

func init() {
	functions.HTTP("RunHTTPServer", app.RunHTTPServer)
}

func main() {
	if os.Getenv("FUNCTION_TARGET") == "" {
		if err := os.Setenv("FUNCTION_TARGET", "RunHTTPServer"); err != nil {
			log.Fatalf("os.Setenv(FUNCTION_TARGET): %v\n", err)
		}
	}

	port := "7202"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	app.Init()

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
