package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"privacy-profile-composer/pkg/composer"
	"privacy-profile-composer/pkg/opa"
	pb "privacy-profile-composer/pkg/proto"
)

var (
	opa_port          = flag.Int("opa_port", 8080, "The OPA server port")
	composer_port     = flag.Int("composer_port", 50051, "The composer server port")
	policy_bundle_dir = flag.String("policy_bundle_dir", "./bundle", "Location of bundle folder with all policies and data")
	compiled_bundle   = flag.String("compiled_bundle", "./bundle.tar.gz", "Location of the compiled bundle")
)

func main() {
	flag.Parse()

	var err error

	err = prepareOpaServer()
	if err != nil {
		panic(err)
	}

	opaServer := registerOpaServer()
	composerServer := registerComposerServer()

	go func() {
		err = http.ListenAndServe(fmt.Sprintf(":%d", *opa_port), opaServer)
		if err != nil {
			log.Fatalf("opa server failed to listen on port %d: %v", opa_port, err)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *composer_port))
	if err != nil {
		log.Fatalf("composer server failed to listen on port %d: %v", composer_port, err)
	}

	if err := composerServer.Serve(lis); err != nil {
		log.Fatalf("composer server failed to serve: %v", err)
	}

}

func prepareOpaServer() error {
	return opa.CompileOPABundle(*policy_bundle_dir, *compiled_bundle)
}

func registerOpaServer() *http.ServeMux {
	s := http.NewServeMux()

	s.HandleFunc("/bundles/bundle.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, *compiled_bundle)
	})

	s.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world!")
	})

	return s
}

func registerComposerServer() *grpc.Server {
	s := grpc.NewServer()

	pb.RegisterPrivacyProfileComposerServer(s, composer.NewComposerServer())

	return s
}
