#!/usr/bin/env bash
# Creates what is missing and updates what exists, so it is safe to run again
# after any change (code or configuration). No bucket: the code goes up as a
# zip straight to Lambda, which keeps its own copy.
#
#   cd backend && deploy/deploy.sh
#
# Needs an authenticated AWS CLI (aws login) and the Firebase key already in
# SSM (see the check below). Undo everything with deploy/teardown.sh.
set -euo pipefail
source "$(dirname "$0")/config.sh"

BUILD=$(mktemp -d)
trap 'rm -rf "$BUILD"' EXIT

if [[ -z $(aws ssm describe-parameters --parameter-filters "Key=Name,Values=$PARAM" --query 'Parameters[].Name' --output text) ]]; then
  echo "SSM parameter $PARAM not found. Put the key there first (the value is never printed):" >&2
  echo "  aws ssm put-parameter --region $REGION --name $PARAM --type SecureString --value file://\$HOME/.config/status-trens-sp/firebase-sa.json" >&2
  exit 1
fi

# --- table -------------------------------------------------------------------
# Provisioned 5/5 stays inside the always-free 25/25 (on-demand has no free
# tier) and leaves room for several lines changing in the same minute.
if ! aws dynamodb describe-table --table-name "$TABLE" >/dev/null 2>&1; then
  echo "creating table $TABLE"
  aws dynamodb create-table --table-name "$TABLE" \
    --attribute-definitions AttributeName=code,AttributeType=N \
    --key-schema AttributeName=code,KeyType=HASH \
    --billing-mode PROVISIONED --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 >/dev/null
  aws dynamodb wait table-exists --table-name "$TABLE"
fi

# --- logs --------------------------------------------------------------------
# Created before the functions so they get a retention instead of the default
# "keep forever".
for fn in "$POLLER" "$API"; do
  aws logs create-log-group --log-group-name "/aws/lambda/$fn" 2>/dev/null || true
  aws logs put-retention-policy --log-group-name "/aws/lambda/$fn" --retention-in-days "$LOG_DAYS"
done

# --- roles -------------------------------------------------------------------
created_role=false
# ensure_role <name> <service principal> <inline policy json>
ensure_role() {
  if ! aws iam get-role --role-name "$1" >/dev/null 2>&1; then
    echo "creating role $1"
    aws iam create-role --role-name "$1" >/dev/null --assume-role-policy-document \
      "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"$2\"},\"Action\":\"sts:AssumeRole\",\"Condition\":{\"StringEquals\":{\"aws:SourceAccount\":\"$ACCOUNT\"}}}]}"
    created_role=true
  fi
  aws iam put-role-policy --role-name "$1" --policy-name "$1" --policy-document "$3"
}

logs_statement() {
  echo "{\"Effect\":\"Allow\",\"Action\":[\"logs:CreateLogStream\",\"logs:PutLogEvents\"],\"Resource\":\"arn:aws:logs:$REGION:$ACCOUNT:log-group:/aws/lambda/$1:*\"}"
}

ensure_role "$POLLER" lambda.amazonaws.com "{\"Version\":\"2012-10-17\",\"Statement\":[
  $(logs_statement "$POLLER"),
  {\"Effect\":\"Allow\",\"Action\":[\"dynamodb:Scan\",\"dynamodb:PutItem\"],\"Resource\":\"arn:aws:dynamodb:$REGION:$ACCOUNT:table/$TABLE\"},
  {\"Effect\":\"Allow\",\"Action\":\"ssm:GetParameter\",\"Resource\":\"arn:aws:ssm:$REGION:$ACCOUNT:parameter$PARAM\"}]}"
ensure_role "$API" lambda.amazonaws.com "{\"Version\":\"2012-10-17\",\"Statement\":[$(logs_statement "$API")]}"
ensure_role "$SCHEDULE" scheduler.amazonaws.com "{\"Version\":\"2012-10-17\",\"Statement\":[
  {\"Effect\":\"Allow\",\"Action\":\"lambda:InvokeFunction\",\"Resource\":\"arn:aws:lambda:$REGION:$ACCOUNT:function:$POLLER\"}]}"
if $created_role; then
  echo "waiting for the new roles to propagate"
  sleep 10
fi

# --- functions ---------------------------------------------------------------
# ensure_function <name> <timeout seconds> <environment json>
ensure_function() {
  echo "building $1"
  mkdir -p "$BUILD/$1"
  CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -trimpath -ldflags="-s -w" \
    -o "$BUILD/$1/bootstrap" "./cmd/${1#"$PREFIX"-}"
  (cd "$BUILD/$1" && zip -q function.zip bootstrap)

  local role="arn:aws:iam::$ACCOUNT:role/$1"
  if ! aws lambda get-function --function-name "$1" >/dev/null 2>&1; then
    echo "creating function $1"
    aws lambda create-function --function-name "$1" --role "$role" \
      --runtime provided.al2023 --architectures arm64 --handler bootstrap \
      --memory-size 128 --timeout "$2" --environment "$3" \
      --zip-file "fileb://$BUILD/$1/function.zip" >/dev/null
    aws lambda wait function-active-v2 --function-name "$1"
  else
    echo "updating function $1"
    aws lambda update-function-code --function-name "$1" --zip-file "fileb://$BUILD/$1/function.zip" >/dev/null
    aws lambda wait function-updated-v2 --function-name "$1"
    aws lambda update-function-configuration --function-name "$1" --role "$role" \
      --memory-size 128 --timeout "$2" --environment "$3" >/dev/null
    aws lambda wait function-updated-v2 --function-name "$1"
  fi
}

# The timeout is half the schedule interval, so a cycle always ends before
# the next one starts.
ensure_function "$POLLER" 30 "{\"Variables\":{\"LINES_TABLE\":\"$TABLE\",\"FIREBASE_SERVICE_ACCOUNT_PARAM\":\"$PARAM\",\"NOTIFICATION_IMAGE_BASE_URL\":\"$IMAGE_BASE_URL\"}}"
ensure_function "$API" 15 '{"Variables":{}}'

# A late or retried cycle is worthless: the next minute polls again anyway.
aws lambda put-function-event-invoke-config --function-name "$POLLER" \
  --maximum-retry-attempts 0 --maximum-event-age-in-seconds 60 >/dev/null

# One poller at a time. New accounts (10 concurrent executions) cannot reserve
# any, since 10 must stay unreserved; this applies once the quota is raised.
if ! aws lambda put-function-concurrency --function-name "$POLLER" --reserved-concurrent-executions 1 >/dev/null 2>&1; then
  echo "note: reserved concurrency not set (account quota too low); the 30 s timeout keeps cycles from overlapping"
fi

# --- function url --------------------------------------------------------------
if ! aws lambda get-function-url-config --function-name "$API" >/dev/null 2>&1; then
  echo "creating function url"
  aws lambda create-function-url-config --function-name "$API" --auth-type NONE >/dev/null
fi
# Public URLs need both permissions since October 2025; an existing
# statement is a conflict, which is fine.
aws lambda add-permission --function-name "$API" --statement-id public-url \
  --action lambda:InvokeFunctionUrl --principal '*' --function-url-auth-type NONE >/dev/null 2>&1 || true
aws lambda add-permission --function-name "$API" --statement-id public-invoke \
  --action lambda:InvokeFunction --principal '*' --invoked-via-function-url >/dev/null 2>&1 || true

# --- schedule ----------------------------------------------------------------
target="{\"Arn\":\"arn:aws:lambda:$REGION:$ACCOUNT:function:$POLLER\",\"RoleArn\":\"arn:aws:iam::$ACCOUNT:role/$SCHEDULE\",\"RetryPolicy\":{\"MaximumRetryAttempts\":0}}"
verb=update-schedule
aws scheduler get-schedule --name "$SCHEDULE" >/dev/null 2>&1 || verb=create-schedule
echo "${verb%-schedule} schedule $SCHEDULE"
aws scheduler "$verb" --name "$SCHEDULE" --schedule-expression 'rate(1 minute)' \
  --flexible-time-window Mode=OFF --state ENABLED --target "$target" >/dev/null

echo
echo "GET /status: $(aws lambda get-function-url-config --function-name "$API" --query FunctionUrl --output text)status"
