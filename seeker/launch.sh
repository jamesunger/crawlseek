#!/bin/sh
sysctl -w net.core.rmem_max=7500000
/usr/local/bin/ipfs init
/bin/watchdog.sh &
nohup /usr/local/bin/ipfs daemon --enable-pubsub-experiment &
sleep 5
ipfsid=$(ipfs id | jq -r '.Addresses[0]')
cat << EOF >> /tmp/redis-input
AUTH crawlseek ${REDIS_PASSWORD}
get frontend-ipfs
EOF

IPFSID=$(cat /tmp/redis-input | redis-cli -u redis://crawlseek:${REDIS_PASSWORD}@${REDIS_HOST} | tail -n 1)
echo "IPFSID: $IPFSID"
ipfs swarm connect $IPFSID
/opt/seeker/bin/seeker
