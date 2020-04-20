#!/usr/bin/env bash

ports="6443 33305 9000"
for x in $ports; do
  </dev/tcp/127.0.0.1/$x
  if [[ $? -eq 0 ]]; then
    echo "probe $x successfully"
  else
    echo "probe $x failed"
    exit 1
  fi
done
