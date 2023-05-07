#!/bin/bash

docker build -f Dockerfile -t sappho:latest ./
docker save sappho:latest > ~/Documents/sappho.tar
microk8s ctr image import ~/Documents/sappho.tar