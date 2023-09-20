#!/bin/bash
if [ $# -lt 1 ]; then
    echo "Usage: $0 <RouterID>"
    exit 1
fi

#run serving go apps
go run serving/main.go $1 &> serving.log &