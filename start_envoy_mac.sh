# #!/bin/bash

# ENVOY=./bazel-bin/source/exe/envoy-static
# CONFIG=newEnvoy.yaml

# PID_FILE=/tmp/envoy.pid

# echo "Starting new Envoy..."

# $ENVOY -c $CONFIG &
# NEW_PID=$!

# echo "New Envoy PID: $NEW_PID"

# # 기존 프로세스 확인
# if [ -f $PID_FILE ]; then
#   OLD_PID=$(cat $PID_FILE)
#   echo "Draining old Envoy PID: $OLD_PID"

#   # graceful drain (SIGTERM)
#   kill -TERM $OLD_PID
# fi

# echo $NEW_PID > $PID_FILE


#!/bin/bash

ENVOY=./bazel-bin/source/exe/envoy-static
CONFIG=newEnvoy.yaml
BASE_ID=1

EPOCH_FILE=/tmp/envoy_epoch
PID_FILE=/tmp/envoy.pid

# epoch 초기화
if [ ! -f $EPOCH_FILE ]; then
  echo 0 > $EPOCH_FILE
fi

OLD_EPOCH=$(cat $EPOCH_FILE)
NEW_EPOCH=$((OLD_EPOCH + 1))

echo "Starting Envoy with epoch $NEW_EPOCH"

$ENVOY \
  -c $CONFIG \
  --restart-epoch $NEW_EPOCH \
  --base-id $BASE_ID \
  --drain-time-s 5 \
  --parent-shutdown-time-s 10 \
  &

NEW_PID=$!
echo "New Envoy PID: $NEW_PID"

# 이전 프로세스 존재하면 종료 트리거
if [ -f $PID_FILE ]; then
  OLD_PID=$(cat $PID_FILE)
  echo "Triggering drain on old Envoy PID: $OLD_PID"

  # Envoy는 SIGTERM 받으면 drain 시작
  kill -TERM $OLD_PID
fi

# 상태 업데이트
echo $NEW_EPOCH > $EPOCH_FILE
echo $NEW_PID > $PID_FILE