#!/bin/bash

#run serving go apps
go run serving/main.go controller &> serving.log &
#run consuming go apps
go run controller/main.go &> controller.log &