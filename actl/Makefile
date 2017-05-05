VERSION=$(shell git describe --tags)

venv: requirements.txt
	virtualenv --python=python3 venv
	venv/bin/pip install -r requirements.txt
