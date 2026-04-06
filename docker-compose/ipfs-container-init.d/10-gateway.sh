#!/bin/sh
set -e
ipfs config --json Gateway.NoFetch true
ipfs config --json Gateway.NoDNSLink true
