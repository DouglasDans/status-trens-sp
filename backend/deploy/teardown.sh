#!/usr/bin/env bash
# Deletes everything deploy.sh created, including the table (the line memory:
# the next deploy starts from a new baseline). The SSM parameter with the
# Firebase key is kept; delete it by hand if wanted.
set -euo pipefail
source "$(dirname "$0")/config.sh"

gone() { echo "deleted $1"; }

aws scheduler delete-schedule --name "$SCHEDULE" 2>/dev/null && gone "schedule $SCHEDULE" || true
aws lambda delete-function-url-config --function-name "$API" 2>/dev/null && gone "function url" || true
for fn in "$POLLER" "$API"; do
  aws lambda delete-function --function-name "$fn" 2>/dev/null && gone "function $fn" || true
done
for role in "$POLLER" "$API" "$SCHEDULE"; do
  aws iam delete-role-policy --role-name "$role" --policy-name "$role" 2>/dev/null || true
  aws iam delete-role --role-name "$role" 2>/dev/null && gone "role $role" || true
done
aws dynamodb delete-table --table-name "$TABLE" >/dev/null 2>&1 && gone "table $TABLE" || true

# A function can still deliver logs right after it is deleted, which would
# recreate its log group without retention: wait, then delete the groups.
sleep 15
for fn in "$POLLER" "$API"; do
  aws logs delete-log-group --log-group-name "/aws/lambda/$fn" 2>/dev/null && gone "log group $fn" || true
done
echo "kept SSM parameter $PARAM (aws ssm delete-parameter --region $REGION --name $PARAM)"
