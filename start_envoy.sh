#!/bin/bash

ENVOY=./bazel-bin/source/exe/envoy-static
CONFIG=envoy.yaml
BASE_ID=1

EPOCH_FILE=/tmp/envoy_epoch

# epoch 초기화
if [ ! -f $EPOCH_FILE ]; then
  echo 0 > $EPOCH_FILE
fi

EPOCH=$(cat $EPOCH_FILE)
NEXT_EPOCH=$((EPOCH + 1))

echo "Starting Envoy with epoch $NEXT_EPOCH"

$ENVOY \
  -c $CONFIG \
  --restart-epoch $NEXT_EPOCH \
  --base-id $BASE_ID \
  --parent-shutdown-time-s 10 \
  --drain-time-s 5 \
  &

echo $NEXT_EPOCH > $EPOCH_FILE