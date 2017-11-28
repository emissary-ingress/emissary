#!python

import sys

output = open(sys.argv[1], "w")

while True:
    line = sys.stdin.readline()

    if not line:
        break

    output.write(line)
    output.flush()

    if line.startswith("try "):
        sys.stdout.write("-")
    else:
        sys.stdout.write(".")

    sys.stdout.flush()

sys.stdout.write("\n")
