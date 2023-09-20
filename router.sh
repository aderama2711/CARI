#!/bin/bash
echo %(date) >> ~/timedate.log &
#run consuming go apps
go run consuming/main.go &> consuming.log &