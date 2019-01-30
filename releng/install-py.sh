#!/usr/bin/env sh

requirements_dev() {
    venv/bin/pip -v install -q -Ur $1
}

requirements_prd() {
    pip3 install -r $1
}

install_dev() {
    venv/bin/pip -v install -q -e $1
}

install_prd() {
    ( cd $1 && python3 setup.py --quiet install )
}

die() {
    echo $*
    exit 1
}

# legal values: dev or prd
MODE=$1
shift
OP=$1
shift

FILES=$*

do_requirements() {
    requirements_${MODE} $1
}

do_install() {
    case "$1" in
        */requirements.txt) install_${MODE} "$(dirname $1)/.";;
    esac
}

for FILE in $FILES; do
    do_${OP} $FILE
done
