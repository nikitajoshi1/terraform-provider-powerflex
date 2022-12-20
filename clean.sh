#!/bin/bash

files=(
    "/.terraform"
    "/.terraform.lock.hcl"
    "/*.tfstate"
    "/*.txt"
    "/*.backup"
)

for d in $(find ./examples -maxdepth 4 -type d)
    do
        for i in "${files[@]}"
            do
                rm -rfv $d$i
            done
        echo "Cleaned {$d}"
    done

echo "Removing sensitive data - ./testdata/powerflex_testing.go"
grep -vs "username" "./testdata/powerflex_testing.go" > tmpfile && mv tmpfile "./testdata/powerflex_testing.go"
grep -vs "password" "./testdata/powerflex_testing.go" > tmpfile && mv tmpfile "./testdata/powerflex_testing.go"
grep -vs "host" "./testdata/powerflex_testing.go" > tmpfile && mv tmpfile "./testdata/powerflex_testing.go"
