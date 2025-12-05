#!/bin/bash

echo "# gotags - better etags-style tags for Go and Python" > README.md
echo >> README.md
go doc | expand -t4 | awk '/^func / { exit } { print }' | \
    while read line; do
        if [[ $line =~ "&&USAGE" ]]; then
            ./gotags -h | unexpand -t2 >> README.md
        else
            echo $line >> README.md
        fi
    done
