package opa

import (
	"bytes"
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/sdk"
	sdktest "github.com/open-policy-agent/opa/sdk/test"
)

func Temp_try_opa_sdk() {
	// code adapted from https://www.openpolicyagent.org/docs/latest/integration/#integrating-with-the-go-sdk
	ctx := context.Background()

	// create a mock HTTP bundle server
	server, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", map[string]string{
		"example.rego": `
				package authz

				import rego.v1

				default allow := false

				allow if input.open == "sesame"
			`,
	}))
	if err != nil {
		fmt.Printf("could not initialize an OPA Bundle API server --- this means that we cannot distribute the target privacy policy to the data plane: ----- %s\n", err)
		return
	}

	defer server.Stop()
	fmt.Printf("initialized an OPA Bundle API server\n")
	// provide the OPA configuration which specifies
	// fetching policy bundles from the mock server
	// and logging decisions locally to the console
	// ----- TODO: Integrate into the Golang filter ------
	// Replace url with http://prose-server.prose-system.svc.cluster.local:8080
	// Remove the leading /bundles/ in the resource // bundles.default.resource=bundle.tar.gz
	config := []byte(fmt.Sprintf(`{
		"services": {
			"bundles": {
				"url": %q
			}
		},
		"bundles": {
			"default": {
				"resource": "/bundles/bundle.tar.gz",
				"polling": {
					"min_delay_seconds": 120,
					"max_delay_seconds": 3600,	
				}
			}
		},
		"decision_logs": {
			"console": true
		}
	}`, server.URL()))

	fmt.Printf("about to instantiate a new opa sdk object\n")
	// create an instance of the OPA object
	opa, err := sdk.New(ctx, sdk.Options{
		ID:     "opa-test-1",
		Config: bytes.NewReader(config),
	})
	fmt.Printf("got a response from sdk.New\n")

	if err != nil {
		fmt.Printf("could not initialize an OPA object --- this means that the data plane cannot evaluate the target privacy policy ----- %s\n", err)
		return
	}

	defer opa.Stop(ctx)
	fmt.Printf("initialized an OPA object\n")

	// get the named policy decision for the specified input
	if result, err := opa.Decision(ctx, sdk.DecisionOptions{Path: "/authz/allow", Input: map[string]interface{}{"open": "sesame"}}); err != nil {
		fmt.Printf("had an error evaluating the policy: %s\n", err)
		return
	} else if decision, ok := result.Result.(bool); !ok || !decision {
		fmt.Printf("result: descision: %v, ok: %v\n", decision, ok)
		return
	} else {
		fmt.Printf("policy accepted the input data \n")
		return
	}
}
