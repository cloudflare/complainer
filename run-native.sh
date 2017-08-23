#!/bin/bash

# This is a wrapper script to run the complainer native binary.
# Compile with "make complainer" and create env.sh first.

RE='^(#.*|[[:space:]]*|_.*)$'

# Reset environment
unset $(grep -vE "${RE}" env.sh | cut -d= -f1)

# Just in case
unset COMPLAINER_MASTERS COMPLAINER_UPLOADER COMPLAINER_REPORTERS COMPLAINER_LOGLEVEL PORT 
unset S3_ACCESS_KEY S3_SECRET_KEY S3_REGION S3_BUCKET S3_PREFIX 
unset SLACK_HOOK_URL SLACK_USERNAME SLACK_CHANNEL SLACK_ICON_EMOJI SLACK_FORMAT FILE_NAME

# Set environment
set -x
. env.sh
export $(grep -vE "${RE}" env.sh | cut -d= -f1)

# Run
./complainer $@
