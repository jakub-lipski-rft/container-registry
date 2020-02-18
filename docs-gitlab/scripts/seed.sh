#!/bin/bash

SEED_REPOSITORIES="${1:-3}"
SEED_TAGS="${2:-3}"

for ((repo=0; repo < SEED_REPOSITORIES; repo++)); do
  for ((tag=0; tag < SEED_TAGS; tag++)); do
    docker build --no-cache -t localhost:5000/repository-${repo}:tag-${tag} -f Dockerfile-Walk . \
      && docker push localhost:5000/repository-${repo}:tag-${tag} \
      && docker rmi localhost:5000/repository-${repo}:tag-${tag}
  done
done
