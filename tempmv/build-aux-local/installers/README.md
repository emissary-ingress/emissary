# Installers

Executables (scripts) in this directory are intended to install and configure additional packages needed by non-OSS modules. AES uses this to add a few features. 

These installers are run

- During `docker build` of the production image as an early step
- From `post-compile.sh` (which is run after every compilation of Go code) repeatedly, to keep the builder container up-to-date

OSS Ambassador doesn't need any installers here because this is the base module: everything it needs to install is listed directly in the Dockerfile.
