#!/bin/bash

cp ../common/* .

export TYPE_MAPPING_PATH=../type_mappings.json && go generate

if [ $? -eq 0 ]
then
  go test
else
echo Failed to generate
  exit 1
fi

rm test.db
./remove-commons.sh