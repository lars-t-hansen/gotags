#!/bin/bash

echo "# gotags - reliable etags-style tag information for Go" > README.md
echo >> README.md
go doc | expand -t4 | awk '/^func / { exit } { print }' | \
    while read line; do
        if [[ $line =~ "&&USAGE" ]]; then
            ./gotags -h | unexpand -t2 >> README.md
        else
            echo $line >> README.md
        fi
    done
