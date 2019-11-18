#!/hint/bash

if [ -e ~/.bashrc ]; then
    # shellcheck disable=SC1090
    . ~/.bashrc
fi

# shellcheck disable=SC1091
. venv/bin/activate
