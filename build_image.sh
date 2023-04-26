#!/bin/bash
docker build -f Dockerfile -t mastodon:latest ./
docker save mastodon:latest > ~/Documents/mastodon.tar
microk8s ctr image import ~/Documents/mastodon.tar