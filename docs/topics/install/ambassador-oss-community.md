# Integration in community projects

**The Ambassador Edge Stack is now available and includes additional functionality beyond the current Ambassador API Gateway.**
These features include automatic HTTPS, the Edge Policy Console UI, OAuth/OpenID Connect authentication support, integrated rate
limiting, a developer portal, and [more](/edge-stack-faq/).

## Ambassador API Gateway integrations

If you still want to use just the Ambassador API Gateway, don't worry! The Ambassador API Gateway
is currently available out-of-the-box in some Kubernetes installers and local environments.

<table style="width:100%">
  <colgroup>
     <col span="1" style="width: 15%;"></col>
     <col span="1" style="width: 85%;"></col>
  </colgroup>

  <thead>
    <tr>
      <th style="text-align:center">Project</th>
      <th>Instructions</th>
    </tr>
  </thead>

  <tbody>
    <tr>
      <td style="text-align:center">
        <a href="https://kind.sigs.k8s.io/" target="_blank">
          <img width="75" src="https://github.com/kubernetes-sigs/kind/blob/master/logo/logo.png?raw=true"></img>
        </a>
      </td>
      <td>
        <a href="https://kind.sigs.k8s.io/docs/user/ingress/#ambassador" target="_blank">KIND</a> documentation.
      </td>
    </tr>
    <tr>
      <td style="text-align:center">
        <a href="https://kubespray.io" target="_blank">
          <img width="75" src="https://kubespray.io/logo/logo-clear.png"></img>
        </a>
      </td>
      <td>
        <a href="https://github.com/kubernetes-sigs/kubespray/tree/master/roles/kubernetes-apps/ingress_controller/ambassador" target="_blank">kubespray</a> README file.
      </td>
    </tr>
    <tr>
      <td style="text-align:center">
        <a href="https://kops.sigs.k8s.io" target="_blank">
          <img width="75" src="https://github.com/kubernetes/kops/raw/master/docs/img/logo-notext.png"></img>
        </a>
      </td>
      <td>
        <a href="https://kops.sigs.k8s.io/operations/addons/#ambassador" target="_blank">KOPS</a> documentation.
      </td>
    </tr>
    <tr>
      <td style="text-align:center">
        <a href="https://minikube.sigs.k8s.io" target="_blank">
          <img width="75" src="https://raw.githubusercontent.com/kubernetes/minikube/master/images/logo/logo.png"></img>
        </a>
      </td>
      <td>
        <a href="https://minikube.sigs.k8s.io/docs/tutorials/ambassador_ingress_controller/" target="_blank">minikube</a> documentation.
      </td>
    </tr>
  </tbody>
</table>
