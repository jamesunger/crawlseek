#!/bin/sh

/usr/local/bin/ipfs init
nohup /usr/local/bin/ipfs daemon --enable-pubsub-experiment &
sleep 5
ipfsid=$(ipfs id | jq -r '.Addresses[0]')
cat << EOF >> /tmp/redis-input
AUTH crawlseek ${REDIS_PASSWORD}
set frontend-ipfs ${ipfsid}
get frontend-ipfs
EOF

cat /tmp/redis-input | redis-cli -u redis://crawlseek:${REDIS_PASSWORD}@${REDIS_HOST}

cd /opt/frontend/html
mkdir results
/opt/frontend/bin/frontend
