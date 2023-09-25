#!/usr/bin/env bash

cleanup() {
    trap - EXIT INT HUP TERM
    # shellcheck disable=2046
    sudo kill -INT $(jobs -p)
    sleep 1
    # shellcheck disable=2046
    sudo kill -9 $(jobs -p)
    killall ffplay
    killall ffmpeg
    killall udp-proxy
}


run_video() {
    ./ffmpeg_play.sh example.sdp > /dev/null 2>&1 &
    ./log_cpu.sh "$DIR/cpu_$NAME.csv" &
    ffmpeg -re -i example_short.mp4 -an -f rtp rtp://127.0.0.1:5004
}

run_udp_message() {
    ./udp.py -l localhost 6004 &
    ./log_cpu.sh "$DIR/cpu_$NAME.csv" &

    for _ in {0..99}; do
        ./udp.py localhost 5004
        sleep 0.5
    done
}


NAME="$1"
shift 1

if [ -z "$NAME" ]; then
    echo "Usage: $0 <name>"
    exit 1
fi

DIR="$PWD"
trap cleanup EXIT INT HUP TERM
cd "$(dirname "$(readlink -f "$0")")" || exit 1

# Measure lateness
# ../udp-wan-proxy -l 5004 -r 6004 -d 1000 --csv "$DIR/proxy_$NAME.csv" "$@" &

# Measure computation delay
../udp-wan-proxy -l 5004 -r 6004 --csv "$DIR/proxy_$NAME.csv" "$@" &

run_udp_message

sleep 1
cleanup
