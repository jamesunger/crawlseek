#!/bin/sh

/usr/local/bin/ipfs init
nohup /usr/local/bin/ipfs daemon --enable-pubsub-experiment &
sleep 5
/opt/seeker/bin/seeker
