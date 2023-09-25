#!/bin/bash
if [ $# -lt 2 ]; then
    echo "Usage: $0 <RouterID> <Prefix>"
    exit 1
fi

#run serving go apps
go run producer/main.go $1 $2 &> serving.log &