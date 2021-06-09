#!/bin/bash

cp ../generator/*.go .
cp ../common/*.go .

export TYPE_MAPPING_PATH=../type_mappings.json && go generate

if [ $? -eq 0 ]
then
  go test -v
else
echo Failed to generate
  exit 1
fi

rm test.db
rm test-no-duplicate.db
rm test-should-migrate.db
./remove-commons.sh