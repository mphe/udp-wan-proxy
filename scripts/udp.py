#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import socket
import sys
from typing import Tuple


def run_listener(s: socket.socket, address: Tuple[str, int]) -> None:
    s.bind(address)

    while True:
        data, _addr = s.recvfrom(4096)

        if not data:
            break

        print(data)


def run_client(s: socket.socket, address: Tuple[str, int], data: str) -> None:
    s.sendto(data.encode("utf-8"), address)


def main() -> int:
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("host", help="Host address")
    parser.add_argument("port", type=int, help="Port")
    parser.add_argument("-l", "--listen", action="store_true", help="Listen mode")
    parser.add_argument("-m", "--message", help="Message to send")
    args = parser.parse_args()

    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    address = (args.host, args.port)

    if args.listen:
        run_listener(s, address)
    else:
        run_client(s, address, args.message or "test")

    return 0


if __name__ == "__main__":
    sys.exit(main())
