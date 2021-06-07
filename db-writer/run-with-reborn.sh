#!/bin/sh

while true
do
  go generate

  if [ $? -eq 0 ]
  then
    go build
  fi

  if [ $? -eq 0 ]
  then
    ./db-writer
  fi

  if [ $? -eq 36 ]
  then
    echo reborn
  else
    break
  fi

done