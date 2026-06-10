package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/m11s-io/terraform-provider-fider/internal/provider"
)

var version string = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider in debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/m11s-io/fider",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err)
	}
}
