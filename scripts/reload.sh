#!/bin/bash

docker ps | grep ronoaldo/ap-5r | awk '{print $1}' | xargs docker stop
sleep 20
docker ps | grep ronoaldo/ap-5r
