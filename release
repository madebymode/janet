#!/bin/bash

cd "${0%/*}" || exit

docker run \
	--rm -ti \
	-v $(pwd):/go/src/github.com/troyxmccall/janet \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $HOME/.config/goreleaser/github_token:/root/.config/goreleaser/github_token \
	-w /go/src/github.com/troyxmccall/janet \
	-e GITHUB_TOKEN \
	janet-build-env $@
