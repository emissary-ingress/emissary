# Product Requirements and Recommendations

Before [installing](../install) the Ambassador Edge Stack, make sure you have, at a very minimum the following:

* A clean, running Kubernetes cluster v1.11 or higher
* `Kubectl`

Then, please review the following recommendations for the Ambassador Edge Stack:

* Port Assignments
* Resource Recommendations
* Version Control
* Requirements for the Edge Policy Console
* VM Recommendations

## Port Assignments

The Ambassador Edge Stack uses the following ports to listen for HTTP/HTTPS traffic automatically via TCP:

| Port | Process | Function |
| :--- | :------ | :------- |
| 8001 | `envoy` | Internal stats, logging, etc.; not exposed outside pod |
| 8002 | `watt`  | Internal `watt` snapshot access; not exposed outside pod |
| 8003 | `ambex` | Internal `ambex` snapshot access; not exposed outside pod |
| 8080 | `envoy` | Default HTTP service port |
| 8443 | `envoy` | Default HTTPS service port |

Note that Ambassador products do not support the UDP protocol.

### Future Port Assignments

Ambassador will never use ports outside of the range of 8000-8999. Third-party software integrating with Ambassador Edge Stack **must not** use ports in this range on the Ambassador pod and will error or fail if they do.

## Resource Recommendations

Because resource usage is expected to be linear with your traffic, we recommend increasing resources for network, CPU, RAM, disk space, bandwidth, as traffic increases.

## Version Control

We recommend that you stay on the latest version of Ambassador. While you can always read back Ambassador's configuration from `annotation`s or its diagnostic service, Ambassador will not do versioning for you.

If you want to be part of the early access releases, learn about how to do so [here](../early-access).

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

## VM Recommendations

If you are using a VM to run the Ambassador Edge Stack, we recommend VM sizes
with 8 or more vCPU, such as D8s-v3 or higher.

These models sizes provide better latency for traffic, while keeping
reconfiguration times under control.

The following table shows the reconfiguration latency of a new mapping in
different VM sizes, measured at the same time the `Mapping` is
applied in the system, until it is effectively available.

| Machine Model    | vCPUs | mem | mbps | Num Nodes | Reconfiguration latency |
|------------------|-------|-----|------|-----------|-------------------------|
| Standard_A4_v2   | 4     | 8   | 1000 | 3         | 5.7s                    |
| Standard_A8_v2   | 8     | 16  | 2000 | 3         | 5.6s                    |
| Standard_D8s_v3  | 8     | 32  | 4000 | 3         | 3.4s                    |
| Standard_DS4_v2  | 8     | 28  | 6000 | 3         | 1.2s                    |
| Standard_F16s_v2 | 16    | 32  | 7000 | 3         | 2.3s                    |
| Standard_E16_v3  | 16    | 128 | 8000 | 3         | 2.3s                    |

[Read more about general purpose VMs](https://docs.microsoft.com/en-us/azure/virtual-machines/sizes-general#av2-series).