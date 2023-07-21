package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/open-policy-agent/opa/compile"
)

func main() {
	f, err := os.Create("./bundle.tar.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	ctx := context.Background()
	compiler := compile.New()
	err = compiler.WithPaths("policy.rego").
		WithOutput(f).
		Build(ctx)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/bundle.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./bundle.tar.gz")
	})

	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8080", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
}
