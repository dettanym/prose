# Random

## How to find envoy version being used by istio

1. Find version of istio-proxy that is being used by a given istio.
   1. E.g. for the version `1.20.3`, there is a `istio.deps` file at the root of the repo. It contains the SHA hash of proxy in use.
2. Open istio/proxy repo at the sha from the previous step.
3. Find version of envoy proxy being used
   1. E.g. for commit `30e213147c5e54158b6176417c39c46eca60c580`, there is a `WORKSPACE` file at the root of the repo that contains the hash of envoy in `ENVOY_SHA` variable.
4. This ENVOY_SHA variable points to the commit inside https://github.com/istio/envoy repository.
   This SHA might or might not exist in original envoyproxy/envoy repository.
5. Install the envoy go dependency of the found sha.
   1. Change into `privacy-profile-composer` folder
   2. Run `go get -u github.com/envoyproxy/envoy@%SHA%`
   3. e.g. the SHA found at step 3 is `bd2cc6e71a9416bd87673ffc79b3245492825d97`, so the command should have `%SHA%` replaced with this value. 
6. The version of golang compiler used by envoy has to match the golang version in our golang filter.
   1. At the commit from previous step, see `go.mod` file and find the version of go compiler in use. In that case it is `1.20`.
