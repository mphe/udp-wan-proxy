#!/usr/bin/env bash

FILE="$1"

if [ -z "$FILE" ]; then
    echo "Missing CSV filepath"
    exit 1
fi

echo "cpu_usage_perc" > "$FILE"

while true; do
    awk '{u=$2+$4; t=$2+$4+$5; if (NR==1){u1=u; t1=t;} else print ($2+$4-u1) * 100 / (t-t1) ; }' \
        <(grep 'cpu ' /proc/stat) <(sleep 1;grep 'cpu ' /proc/stat) >> "$FILE"
done
