#!/bin/bash

go test -v -timeout 5s | tee log.txt
