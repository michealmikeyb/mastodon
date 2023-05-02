#!/bin/bash

docker build -f Dockerfile -t sappho:latest ./
docker save sappho:latest > ~/Documents/sappho.tar
microk8s ctr image import ~/Documents/sappho.tar
docker run -p 8080:8080 --env DB_HOST=host.docker.internal --env DB_USER --env DB_PORT --env DB_PASS --env DB_NAME sappho:latest