package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
	providerversion "github.com/rixlhq/terraform-provider-netcup/internal/version"
)

var (
	version string = "dev"
)

func main() {
	if version == "" || version == "dev" {
		version = providerversion.Version
	}
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/rixlhq/netcup",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
