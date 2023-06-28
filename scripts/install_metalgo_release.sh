#!/usr/bin/env bash
set -e

# Load the versions
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

VERSION=$DIJETSNODE_VERSION

############################
# download dijetsnode
# https://github.com/lasthyphen/dijetsnode/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BASEDIR=${BASE_DIR:-"/tmp/dijetsnode-release"}
mkdir -p ${BASEDIR}
AVAGO_DOWNLOAD_URL=https://github.com/lasthyphen/dijetsnode/releases/download/${VERSION}/dijetsnode-linux-${GOARCH}-${VERSION}.tar.gz
AVAGO_DOWNLOAD_PATH=${BASEDIR}/dijetsnode-linux-${GOARCH}-${VERSION}.tar.gz
if [[ ${GOOS} == "darwin" ]]; then
  AVAGO_DOWNLOAD_URL=https://github.com/lasthyphen/dijetsnode/releases/download/${VERSION}/dijetsnode-macos-${VERSION}.zip
  AVAGO_DOWNLOAD_PATH=${BASEDIR}/dijetsnode-macos-${VERSION}.zip
fi

DIJETSNODE_BUILD_PATH=${DIJETSNODE_BUILD_PATH:-${BASEDIR}/dijetsnode-${VERSION}}
mkdir -p $DIJETSNODE_BUILD_PATH

if [[ ! -f ${AVAGO_DOWNLOAD_PATH} ]]; then
  echo "downloading dijetsnode ${VERSION} at ${AVAGO_DOWNLOAD_URL} to ${AVAGO_DOWNLOAD_PATH}"
  curl -L ${AVAGO_DOWNLOAD_URL} -o ${AVAGO_DOWNLOAD_PATH}
fi
echo "extracting downloaded dijetsnode to ${DIJETSNODE_BUILD_PATH}"
if [[ ${GOOS} == "linux" ]]; then
  mkdir -p ${DIJETSNODE_BUILD_PATH} && tar xzvf ${AVAGO_DOWNLOAD_PATH} --directory ${DIJETSNODE_BUILD_PATH} --strip-components 1
elif [[ ${GOOS} == "darwin" ]]; then
  unzip ${AVAGO_DOWNLOAD_PATH} -d ${DIJETSNODE_BUILD_PATH}
  mv ${DIJETSNODE_BUILD_PATH}/build/* ${DIJETSNODE_BUILD_PATH}
  rm -rf ${DIJETSNODE_BUILD_PATH}/build/
fi

DIJETSNODE_PATH=${DIJETSNODE_BUILD_PATH}/dijetsnode
DIJETSNODE_PLUGIN_DIR=${DIJETSNODE_BUILD_PATH}/plugins

echo "Installed DIJETSNODE release ${VERSION}"
echo "DIJETSNODE Path: ${DIJETSNODE_PATH}"
echo "Plugin Dir: ${DIJETSNODE_PLUGIN_DIR}"
