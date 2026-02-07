#!/bin/bash -xe

cd web/frontend
npm run build

cd ../..
go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

./ms-mqtt-adapter
