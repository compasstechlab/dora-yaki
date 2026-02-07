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
	port := "7202"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	app.Init()

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
