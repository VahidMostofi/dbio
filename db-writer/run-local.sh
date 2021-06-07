#!/bin/bash
set -a
source dev.env
cp ../common/* .

go generate

if [ $? -eq 0 ]
then
  go build
else
  exit 1
fi

if [ $? -eq 0 ]
then
  ./db-writer
else
  exit 1
fi
set +a