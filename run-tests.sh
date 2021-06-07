#!/bin/bash

if [ -f "common/run-tests.sh" ]; then
  cd common
  ./run-tests.sh
  cd ..
fi


if [ -f "db-writer/run-tests.sh" ]; then
  cd db-writer
  ./run-tests.sh
  cd ..
fi

if [ -f "db-reader/run-tests.sh" ]; then
  cd db-reader
  ./run-tests.sh
  cd ..
fi

