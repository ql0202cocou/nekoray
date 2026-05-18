#!/bin/bash
set -e

source libs/env_deploy.sh
ENV_NEKORAY=1
source libs/get_source_env.sh
pushd ..

####

if [ ! -d "sing-box" ]; then
  git clone --no-checkout https://github.com/MatsuriDayo/sing-box.git
fi
pushd sing-box
git fetch origin "$COMMIT_SING_BOX"
git checkout "$COMMIT_SING_BOX"

popd

####

if [ ! -d "sing-quic" ]; then
  git clone --no-checkout https://github.com/SagerNet/sing-quic.git
else
  git -C sing-quic remote set-url origin https://github.com/SagerNet/sing-quic.git
fi
pushd sing-quic
git fetch origin "$COMMIT_SING_QUIC"
git checkout "$COMMIT_SING_QUIC"

popd

####

if [ ! -d "libneko" ]; then
  git clone --no-checkout https://github.com/MatsuriDayo/libneko.git
fi
pushd libneko
git fetch origin "$COMMIT_LIBNEKO"
git checkout "$COMMIT_LIBNEKO"

popd

####

popd
