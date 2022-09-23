#!/bin/bash


brew upgrade
brew install redis

# run redis docker image
docker run --rm --name redis  -p 6379:6379 -d redis
