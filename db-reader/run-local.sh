#!/bin/bash
set -a
source dev.env
cp ../generator/*.go .
cp ../common/*.go .

go generate

if [ $? -eq 0 ]
then
  go build
else
  exit 1
fi

if [ $? -eq 0 ]
then
  ./db-reader
else
  exit 1
fi
set +a