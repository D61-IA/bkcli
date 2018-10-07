#!/bin/bash

BASEDIR=$(dirname "$0")
VERSION="$(cat ${BASEDIR}/../VERSION)"
IMAGENAME="stellargraph/bkcli"

docker build -t ${IMAGENAME}:"${VERSION}" -f "${BASEDIR}"/Dockerfile "${BASEDIR}/../"
