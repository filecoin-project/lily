#!/bin/bash

# Make cURL request to retrieve JSON response
response=$(curl "https://secretmanager.googleapis.com/v1/projects/protocol-labs-data/secrets/LILY_ENV_FILE/versions/latest:access" \
    --request "GET" \
    --header "authorization: Bearer $(gcloud auth print-access-token)" \
    --header "content-type: application/json")

# Extract the `.payload.data` field from the response JSON
data=$(echo $response | jq -r '.payload.data')

# Decode the `data` field using base64
echo $data | base64 -d > .env

