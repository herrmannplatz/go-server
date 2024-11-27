#!/bin/bash

base64url_encode() {
  read input
  echo -n "$input" | base64 | tr -d '=' | tr '+/' '-_'
}

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <secret>"
    exit 1
fi

secret=$1

HEADER=$(echo -n "{\"alg\":\"HS256\",\"typ\":\"JWT\"}" | base64url_encode)
PAYLOAD=$(echo -n "{\"iat\":$(date +%s)}" | base64url_encode)
SIGNATURE=$(echo -n "$header_base64.$PAYLOAD" | openssl dgst -binary -sha256 -hmac "$secret" | base64url_encode)

echo "$HEADER.$PAYLOAD.$SIGNATURE"

