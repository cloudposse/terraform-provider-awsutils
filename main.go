package main

import (
	"context"
	"flag"
	"log"

	"github.com/cloudposse/terraform-provider-awsutils/internal/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
)


func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	serverFactory, err := provider.ProtoV5ProviderServerFactory(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt

	if *debugFlag {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	logFlags := log.Flags()
	logFlags = logFlags &^ (log.Ldate | log.Ltime)
	log.SetFlags(logFlags)

	err = tf5server.Serve(
		"registry.terraform.io/cloudposse/awsutils",
		serverFactory,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
