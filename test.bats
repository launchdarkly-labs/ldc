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

DEFAULT_CONFIG="test"
DEFAULT_PROJECT="ldc-test"
DEFAULT_ENV="production"

DATE=$(date +%Y-%m-%d)
FLAG_KEY=$(echo ldc-${DATE}-${RANDOM} | cut -c-20)
PROJ_KEY="$FLAG_KEY"
ENV_KEY="$FLAG_KEY"
GOAL_NAME="$FLAG_KEY"
PROJECT_NAME="ldc-test"

@test "smoke test for setting up goals on flag" {
    $COMMAND flags create "$FLAG_KEY" "ldc at $DATE"
    $COMMAND flags on "$FLAG_KEY"
#    $COMMAND flags rollout "$FLAG_KEY" 0:true:50 1:false:50
    $COMMAND flags off "$FLAG_KEY"

    $COMMAND goals create custom "$GOAL_NAME" event-key
    $COMMAND goals attach "$GOAL_NAME" "$FLAG_KEY"
    $COMMAND goals detach "$GOAL_NAME" "$FLAG_KEY"
    $COMMAND goals delete "$GOAL_NAME"

    $COMMAND flags delete "$FLAG_KEY"
}

function testFlagAtPath() {
    local path="$1"
    echo "Creating flag at $path"
    $COMMAND flags create "$path" "ldc at $DATE"
    echo "Deleting flag at $path"
    $COMMAND flags delete "$path"
}

@test "flag paths" {
    testFlagAtPath "$FLAG_KEY"
    testFlagAtPath "/$DEFAULT_PROJECT/$FLAG_KEY"
    testFlagAtPath "/.../$FLAG_KEY"
    testFlagAtPath "//$DEFAULT_CONFIG/$DEFAULT_PROJECT/$FLAG_KEY"
    testFlagAtPath "//$DEFAULT_CONFIG/.../$FLAG_KEY"
}

function testGoalAtPath() {
    local path="$1"
    echo "Creating custom goal at $path"
    $COMMAND goals create custom "$path" "ldc at $DATE"
    echo "Deleting goal at $path"
    $COMMAND goals delete "$path"
}

@test "goal paths" {
    testGoalAtPath "$GOAL_NAME"
    testGoalAtPath "/$DEFAULT_PROJECT/$DEFAULT_ENV/$GOAL_NAME"
    testGoalAtPath "//$DEFAULT_CONFIG/$DEFAULT_PROJECT/$DEFAULT_ENV/$GOAL_NAME"
    testGoalAtPath "/.../.../$GOAL_NAME"
    testGoalAtPath "//$DEFAULT_CONFIG/.../.../$GOAL_NAME"
}

function testEnvironmentAtPath() {
    local path="$1"
    echo "Creating environment at $path"
    $COMMAND environments create "$path" "ldc at $DATE"
    echo "Deleting environment at $path"
    $COMMAND environments delete "$path"
}

@test "environment paths" {
    testEnvironmentAtPath "$ENV_KEY"
    testEnvironmentAtPath "/$DEFAULT_PROJECT/$ENV_KEY"
    testEnvironmentAtPath "//$DEFAULT_CONFIG/$DEFAULT_PROJECT/$ENV_KEY"
    testEnvironmentAtPath "/.../$ENV_KEY"
    testEnvironmentAtPath "//$DEFAULT_CONFIG/.../$ENV_KEY"
}

function testProjectAtPath() {
    local path="$1"
    echo "Creating project at $path"
    $COMMAND projects create "$path" "ldc at $DATE"
    echo "Deleting project at $path"
    $COMMAND projects delete "$path"
}

@test "project paths" {
    testProjectAtPath "$PROJ_KEY"
    testProjectAtPath "/$PROJ_KEY"
    testProjectAtPath "//$DEFAULT_CONFIG/$PROJ_KEY"
}