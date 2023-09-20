#!/bin/bash
echo $(date) >> ~/timedate.log &
#run consuming go apps
go run controller/main.go &> controller.log &