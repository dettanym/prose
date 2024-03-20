package opa

import (
	"context"
	"os"

	"github.com/open-policy-agent/opa/compile"
)

func CompileOPABundle(policyBundleDir string, bundle string) error {
	f, err := os.Create(bundle)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx := context.Background()
	compiler := compile.New()

	err = compiler.
		WithPaths(policyBundleDir).
		WithOutput(f).
		Build(ctx)

	return err
}
