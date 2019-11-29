# Product Requirements and Recommendations

Before [installing](/user-guide/install) the Ambassador Edge Stack, make sure you have, at a very minimum the following:

* A Clean, running Kubernetes cluster v1.11 or higher
* Kubectl

Then, please review the following recommendations for the Ambassador Edge Stack:

* Port Assignments
* Resource Recommendations
* Version Control
* Requirements for the Edge Policy Console

## Port Assignments

The Ambassador Edge Stack uses the following ports to listen for HTTP/HTTPS traffic automatically via TCP:

| Port | Process | Function |
| :--- | :------ | :------- |
| 8001 | `envoy` | Internal stats, logging, etc.; not exposed outside pod |
| 8002 | `watt`  | Internal `watt` snapshot access; not exposed outside pod |
| 8003 | `ambex` | Internal `ambex` snapshot access; not exposed outside pod |
| 8080 | `envoy` | Default HTTP service port |
| 8443 | `envoy` | Default HTTPS service port |
| 8877 | `diagd` | Direct access to diagnostics UI |
| 8500 | ???

Ambassador binds to ports 8080, 8443, and 8500 by default. Note that Ambassador products do not support the UDP protocol.

### Future Port Assignments

Any future assignments you must make for Ambassador must come from the port range 8000-8499.

Third-party software integrating with Ambassador Edge Stack should not use ports in this range on the Ambassador pod.

## Resource Recommendations

Because resource usage is expected to be linear with your traffic, we recommend increasing resources for network, CPU, RAM, disk space, bandwidth, as traffic increases.

## Version Control

We recommend that you stay on the latest version of Ambassador.

If you want to be part of the early access releases, learn about how to do so [here](/user-guide/early-access).

## Edge Policy Console Requirements

To use the administrative interface, the Edge Policy Console, we recommend that you use the following operating systems: 

* Linux (x84 64bit)
* OS X 10.11 (El Capitan) or newer
* Windows 8 or newer

Note that Linux installs do not support ARMS, MIPS, or 32bit.

Additionally, `edgectl` is not supported on Windows and prevents you from using the Console in a browser. However, you can use the command line.

The Edge Policy Console will work best in the following browsers:

* Firefox v63+
* Chrome 61+
* Safari 11+
* Opera 41
