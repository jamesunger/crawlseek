#!/bin/sh
sysctl -w net.core.rmem_max=7500000
/usr/local/bin/ipfs init
nohup /usr/local/bin/ipfs daemon --enable-pubsub-experiment &
sleep 5
/opt/seeker/bin/seeker
