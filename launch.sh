#!/bin/sh

/usr/local/bin/ipfs init
nohup /usr/local/bin/ipfs daemon --enable-pubsub-experiment &
sleep 10
cd /opt/frontend/html
mkdir results
/opt/frontend/bin/frontend
