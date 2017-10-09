#!/bin/bash

while true ; do
        docker ps | grep gcr.io/ronoaldoconsulting/ap-5r | awk '{print $1}' | xargs docker logs --tail 100 -f
        echo "*** TERMINATED *** Reloading ..."
        sleep 4
done
