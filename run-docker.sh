#!/bin/bash

# This is a wrapper script to run the complainer docker container. 
# Compile with "make docker" and create env.sh first.

RE='^(#.*|[[:space:]]*|_.*)$'

# Set environment
set -x
. env.sh
set +x
export $(grep -vE "${RE}" env.sh | cut -d= -f1)

# Run docker
PORT_ARG=""
if [ \! -z "${PORT}" ]; then
    PORT_ARG="--publish ${_DOCKER_HOST_PORT}:${PORT}"
fi

ENV_ARG=""
for var in $(grep -vE "${RE}" env.sh | cut -d= -f1) ; do
    ENV_ARG="${ENV_ARG} -e ${var} "
done

VOL_ARG=""
if [ \! -z "${_DOCKER_HOST_VAR}" ]; then
    mkdir ${_DOCKER_HOST_VAR}
    VOL_ARG="--volume ${_DOCKER_HOST_VAR}:/var/log/mesos-complainer"
fi

set -x
docker run \
       ${ENV_ARG} \
       ${PORT_ARG} \
       ${VOL_ARG} \
       --rm \
       $(DOCKER_ACCOUNT=$_DOCKER_ACCOUNT make print-docker-repo-tag)
