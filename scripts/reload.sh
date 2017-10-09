#!/bin/bash

docker ps | grep gcr.io/ronoaldoconsulting/ap-5r | awk '{print $1}' | xargs docker stop
sleep 20
docker ps | grep gcr.io/ronoaldoconsulting/ap-5r
