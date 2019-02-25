#!/usr/bin/env bats

function setup() {
    CLEANUP="echo Cleaning up"
    CONFIG_DIR=$(mktemp -d)
    pushd "$CONFIG_DIR"
    CONFIG_FILE=ldc.json
    cat > "$CONFIG_FILE" <<EOF
{
  "test": {
    "apitoken": "$TEST_API_TOKEN",
    "defaultproject": "ldc-test",
    "defaultenvironment": "production"
  }
}
EOF
    COMMAND="$LDC --config test --config-file $CONFIG_FILE"
}

function teardown() {
    eval "$CLEANUP"
    rm "$CONFIG_FILE"
    popd
    rmdir "$CONFIG_DIR"
}

DATE=$(date +%Y-%m-%d)
FLAG_KEY=$(echo ldc-${DATE}-${RANDOM} | cut -c-20)
PROJECT_NAME="ldc-test"

@test "smoke test for basic flag operations" {
    $COMMAND flags create "$FLAG_KEY" "ldc at $DATE"
    CLEANUP="$CLEANUP; $LDC flags delete \"$FLAG_KEY\""
    $COMMAND flags on "$FLAG_KEY"
    $COMMAND flags rollout "$FLAG_KEY" 0:true:50 1:false:50
    $COMMAND flags off "$FLAG_KEY"
}
