# Shared by deploy.sh and teardown.sh. Names only: no secret lives here.
REGION=${AWS_REGION:-sa-east-1}
PREFIX=status-trens-sp
POLLER=$PREFIX-poller
API=$PREFIX-api
SCHEDULE=$PREFIX-poll
TABLE=$PREFIX-lines
PARAM=/status-trens-sp/firebase-sa
IMAGE_BASE_URL=https://raw.githubusercontent.com/DouglasDans/status-trens-sp/main/backend/badges
LOG_DAYS=14

aws() { command aws --region "$REGION" "$@"; }
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
