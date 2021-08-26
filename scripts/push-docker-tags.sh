#!/usr/bin/env bash

# push-docker-tags.sh
#
# Run from ci to tag images based on the current branch or tag name.
# Like dockerhub autobuild config, but somewhere we can version control it.
#
# The `docker-push` job in .circleci/config.yml runs this script to decide
# what tag, if any, to push to dockerhub.
#
# Usage:
#   ./push-docker-tags.sh <image name> <git commit sha1> <docker tag> [dry run]
#
# Example:
#   # dry run. pass a 5th arg to have it print what it would do rather than do it.
#   ./push-docker-tags.sh myiamge testingsha "" dryrun
#
#   # push tag for a new release tag
#   ./push-docker-tags.sh myimage testingsha v0.5.0
#
#   # serving suggestion in circle ci - https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables
#   ./push-docker-tags.sh filecoin/lily $CIRCLE_SHA1 $CIRCLE_TAG
#
set -euo pipefail

if [[ $# -lt 3 ]] ; then
  echo 'At least 3 args required. Pass 4 args for a dry run.'
  echo 'Usage:'
  echo './push-docker-tags.sh <image name> <git commit sha1> <docker tag> [dry run]'
  exit 1
fi

IMAGE_NAME=$1
GIT_SHA1=$2
GIT_SHA1_SHORT=$(echo "$GIT_SHA1" | cut -c 1-8)
GIT_TAG=${3:-""}
DRY_RUN=${4:-false}

pushTag () {
  local IMAGE_TAG="${1//\//-}"
  if [ "$DRY_RUN" != false ]; then
    echo "DRY RUN!"
    echo docker tag "$IMAGE_NAME" "$IMAGE_NAME:$IMAGE_TAG"
    echo docker push "$IMAGE_NAME:$IMAGE_TAG"
  else
    echo "Tagging $IMAGE_NAME:$IMAGE_TAG and pushing to dockerhub"
    docker tag "$IMAGE_NAME" "$IMAGE_NAME:$IMAGE_TAG"
    docker push "$IMAGE_NAME:$IMAGE_TAG"
  fi
}

pushTag "${GIT_TAG}"
