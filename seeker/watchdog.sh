#!/bin/bash


while true; do
        RSS=$(ps -p `pidof ipfs` -o rss | tail -n 1)
        echo "$RSS"
        if (( $RSS >= 2000000 )); then
                echo kill `pidof ipfs`
        fi
	sleep 5
done
