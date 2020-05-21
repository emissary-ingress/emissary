import './index.less'

# Installing the Ambassador Edge Stack
<div id="index-installContainer">
<span id="index-installContainerText">The Ambassador Edge Stack can be installed many different ways.</span><span>&nbsp;&nbsp;</span>
<div class="index-dropdown">
  <button class="index-dropBtn">Jump to Installation Type</button>
  <div class="index-dropdownContent">
    <a href="#index-installKubernetesYaml">Kubernetes YAML</a>
    <a href="#index-installHelm">Helm</a>
    <a href="#index-installDocker">Docker</a>
    <a href="#index-installAmbassadorOperator">Ambassador Operator</a>
    <a href="#index-installBareMetal">Bare Metal</a>
    <a href="#index-installUpgrade">Upgrade</a>
  </div>
</div>
</div>

## <img class="os-logo" src="../../images/kubernetes.png"/> Install via Kubernetes YAML 
Kubernetes via YAML is the most common approach to install Ambassador Edge Stack,
especially in production environments, with our default, customizable manifest.
So if you want complete configuration control over specific parameters of your
installation, use the [manual YAML installation method](yaml-install).
<p id="index-installHelm"></p><br/>

## <img class="os-logo" src="../../images/helm-navy.png"/> Install via Helm 
Helm, the package manager for Kubernetes, is another popular way to install
Ambassador Edge Stack through the pre-packaged Helm chart. Full details, including
the differences for Helm 2 and Helm3, are in the [Helm instructions.](helm/)
<p id="index-installDocker"></p><br/>

## <img class="os-logo" src="../../images/docker.png"/> Install Locally on Docker 
The Docker install will let you try the Ambassador Edge Stack locally in seconds, 
but is not supported for production workloads. [Try Ambassador on Docker.](docker/)
<p id="index-installAmbassadorOperator"></p><br/>

## Install via the Ambassador Operator
The Ambassador Edge Stack Operator automates installs (day 1 operations) and
updates (day 2 operations), among other actions. To use the powerful Ambassador
Operator, [follow the Ambassador Edge Stack Operator instructions](aes-operator).
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
