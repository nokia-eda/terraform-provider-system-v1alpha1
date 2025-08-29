package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/nokia/eda/apps/terraform-provider-system/internal/provider"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "github.com/nokia-eda/system-v1alpha1",
	}

	err := providerserver.Serve(context.Background(), provider.New("v1alpha1"), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
