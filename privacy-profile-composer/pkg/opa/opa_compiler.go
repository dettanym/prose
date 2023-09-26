package opa

import (
	"context"
	"os"

	"github.com/open-policy-agent/opa/compile"
)

func CompileOPABundle(policy_file string, bundle string) error {
	f, err := os.Create(bundle)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx := context.Background()
	compiler := compile.New()

	err = compiler.
		WithPaths(policy_file).
		WithOutput(f).
		Build(ctx)

	return err
}
