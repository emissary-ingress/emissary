import './index.less'

# Installing the Ambassador Edge Stack
<div id="index-installContainer">
<span id="index-installContainerText">The Ambassador Edge Stack can be installed many different ways.</span><span>&nbsp;&nbsp;</span>
<div class="index-dropdown">
  <button class="index-dropBtn">Jump to Installation Type</button>
  <div class="index-dropdownContent">
    <a href="#index-installKubernetesYaml">Kubernetes YAML</a>
    <a href="#index-installHelm">Helm</a>
    <a href="#index-installMac">Mac</a>
    <a href="#index-installLinux">Linux</a>
    <a href="#index-installWindows">Windows</a>
    <a href="#index-installAmbassadorOperator">Ambassador Operator</a>
    <a href="#index-installDocker">Docker</a>
    <a href="#index-installBareMetal">Bare Metal</a>
    <a href="#index-installUpgrade">Upgrade</a>
  </div>
</div>
</div>

<p id="index-installKubernetesYaml"></p><br/>

## Install via Kubernetes YAML

Kubernetes via YAML is the most common approach to install Ambassador Edge Stack,
especially in production environments, with our default, customizable manifest.
So if you want complete configuration control over specific parameters of your
installation, use the [manual YAML installation method](yaml-install).
<p id="index-installHelm"></p><br/>

## Install via Helm
[![Helm](../../images/helm.png)](helm/)

Helm, the package manager for Kubernetes, is another popular way to install
Ambassador Edge Stack through the pre-packaged Helm chart. Full details, including
the differences for Helm 2 and Helm3, are in the [Helm instructions.](helm/)

<span id="index-installMac"></span><br/>

## Install from MacOS <img class="os-logo" src="../../images/apple.png"/>
1. (1a) [Download the `edgectl` installer](https://metriton.datawire.io/downloads/darwin/edgectl) 
 or (1b) download it with a curl command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl
    ```

    If you decide to download the file with (1b), you may encounter a security block. To continue, use this procedure:
    * Go to **System Preferences > Security & Privacy > General**.
    * Click the **Open Anyway** button.
    * On the new dialog, click the **Open** button.

2. Run the installer with `edgectl install`

3. The installer will provision a load balancer, configure TLS, 
and provide you with an `edgestack.me` subdomain. The `edgestack.me` subdomain 
allows the Ambassador Edge Stack to automatically provision TLS and HTTPS
for a domain name, so you can get started right away.
<span id="index-installLinux"></span><br/>

## Install from Linux <img class="os-logo" src="../../images/linux.png"/> 

1. (1a) [Download the `edgectl` installer](https://metriton.datawire.io/downloads/linux/edgectl) or
 (1b) download it with a curl
   command:

    ```shell
    sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl
    ```
2. Run the installer with `edgectl install`

3. The installer will provision a load balancer, configure TLS, 
and provide you with an `edgestack.me` subdomain. The `edgestack.me` subdomain 
allows the Ambassador Edge Stack to automatically provision TLS and HTTPS
for a domain name, so you can get started right away.
<p id="index-installWindows"></p><br/>

## Install from Windows <img class="os-logo" src="../../images/windows.png"/>

1. [Download the `edgectl.exe` installer](https://metriton.datawire.io/downloads/windows/edgectl.exe).
2. Run the installer with `edgectl install`
3. The installer will provision a load balancer, configure TLS, 
and provide you with an `edgestack.me` subdomain. The `edgestack.me` subdomain 
allows the Ambassador Edge Stack to automatically provision TLS and HTTPS
for a domain name, so you can get started right away.
<p id="index-installAmbassadorOperator"></p><br/>

## Install via the Ambassador Operator

The Ambassador Edge Stack Operator automates installs (day 1 operations) and
updates (day 2 operations), among other actions. To use the powerful Ambassador
Operator, [follow the Ambassador Edge Stack Operator instructions](aes-operator).
<p id="index-installDocker"></p><br/>

## Install Locally on Docker
[![Docker](../../images/docker.png)](docker/)

The Docker install will let you try the Ambassador Edge Stack locally in seconds, 
but is not supported for production workloads. [Try Ambassador on Docker.](docker/)
<p id="index-installBareMetal"></p><br/>

## Install on Bare Metal

If you don't have a load balancer in front of your Kubernetes, the Bare Metal 
installation mechanism can still be used to expose the Ambassador Edge Stack. 
We've got [instructions for bare metal installations] including exposing 
the Ambassador Edge Stack via a NodePort or the host network.
<p id="index-installUpgrade"></p><br/>

## Upgrade Options

If you already have an existing installation of the Ambassador Edge Stack or
Ambassador API Gateway, you can upgrade your instance:

1. [Upgrade to the Ambassador Edge Stack from the API Gateway](upgrade-to-edge-stack/).
2. [Upgrade your Ambassador Edge Stack instance](upgrading/) to the latest version.

# What’s Next?

The Ambassador Edge Stack has a comprehensive range of [features](/features/) to
support the requirements of any edge microservice. To learn more about how the
Ambassador Edge Stack works, along with use cases, best practices, and more,
check out the [Welcome page](../../) or read the [Ambassador
Story](../../about/why-ambassador).
