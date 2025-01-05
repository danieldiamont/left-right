#!/bin/bash

go test -v -timeout 10s | tee log.txt
