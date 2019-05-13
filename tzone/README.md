This directory is self contained and the Dockerfile hardcodes the
envoy base image. This is all for illustration/poc purposes.

Step 1:

  While in this directory, run:

    docker build . -t tzone

Step 2:

  While anywhere, run:

    docker run --cap-add NET_ADMIN --dns-search . -it tzone

  This will start a shell inside the `tzone` container.  The
  `--cap-add NET_ADMIN` gives the docker container the priviliges
  needed to run `iptables` inside a container.

Step 3:

  While in the tzone shell, run:

    ./run-stuff.sh

  The `./run-stuff.sh` script runs `teleproxy` and the `kat-backend`
  in the background. It sends the output to `/tmp/teleproxy.log` and
  `/tmp/backend.log` respectively. Just cat the script for details,
  it's very simple.

Step 4:

  While in the tzone shell, run:

    ./load-routes.sh

  This will run a curl command that loads the routes listed the
  `routes.json` file into teleproxy. You can edit `routes.json` and
  rerun the script to reload the routes.

Step 5:

  While in the tzone shell, run:

    host hi
    host ho
    host hello

  Note that all these names resolve to whatever ip address is supplied
  in routes.json.

Step 6:

  While in the tzone shell, run:

    curl hi
    curl ho
    curl hello

  Note that all these requests go to the one kat-backend process that
  is listening on port 8080.
