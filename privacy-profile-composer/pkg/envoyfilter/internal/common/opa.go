package common

import (
	"bytes"
	"context"

	"github.com/open-policy-agent/opa/sdk"
)

var GlobalAuthAgent *sdk.OPA

func init() {
	opaObj, err := sdk.New(context.Background(), sdk.Options{
		ID: "golang-filter-opa",
	})

	if err != nil {
		panic(err)
	}

	GlobalAuthAgent = opaObj
}

func UpdateAuthAgent(opaConfig string) (*sdk.OPA, error) {
	err := GlobalAuthAgent.Configure(context.Background(), sdk.ConfigOptions{
		Config: bytes.NewReader([]byte(opaConfig)),
	})

	return GlobalAuthAgent, err
}
