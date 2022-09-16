#!/bin/sh
export HOME="$0.d/home"
export PATH="$0.d/bin:$0.d/venv/bin:$PATH"
exec ct "$@"
