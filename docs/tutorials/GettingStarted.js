import React, { Component } from "react";
import CopyButton from '../../../../src/components/CodeBlock/CopyButton';
import './getting-started.less';

class GettingStarted extends Component {
  componentDidMount() {
    var os = "other";
    if (/Mac(intosh|Intel|PPC|68K)/.test(window.navigator.platform)) {
      os = 'mac';
    } else if (/Win(dows|32|64|CE)/.test(window.navigator.platform)) {
      os = 'windows';
    } else if (/Linux/.test(window.navigator.platform)) {
      os = 'linux';
    }

    function renderHeader() {
      switch (os) {
        case "mac":
          document.getElementById("QS-showMac").style.display = "inline-block";
          document.getElementById("QS-showMacAside3").style.display = "inline-block";
          break;
        case "linux":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside3").style.display = "inline-block";
          break;
        case "windows":
          document.getElementById("QS-showWindows").style.display = "inline-block";
          document.getElementById("QS-showWindowsAside3").style.display = "inline-block";
          break;        
        case "other":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside3").style.display = "inline-block";
      }
    };
    renderHeader(os);
  }

  render() {
    return (
    <div className="QS-grid">

      <div className="QS-wrapper">
        <span className="QS-k8"><img className="QS-k8Logo" src="../../images/kubernetes.png"/></span>
      </div>

      <div className="QS-aside QS-aside1">
        <div className="QS-asideText">
          <ul id="QS-asideBullets">
            <li>Have Kubernetes? Deploy Ambassador Edge Stack with yaml:</li>
              <div className="styles-module--CodeBlock--1UB4s">
                <div className="QS-codeblockInstall QS-Aside1-codeblockInstall">
                <span className="QS-copyButton"><CopyButton content="
                kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml && \kubectl wait --for condition=established --timeout=90s crd -lproduct=aes && \kubectl apply -f https://www.getambassador.io/yaml/aes.yaml && \kubectl -n ambassador wait --for condition=available --timeout=90s deploy -lproduct=aes">Copy</CopyButton></span>
                  <div className="token-line">
                    <span className="token-plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">apply</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-f https://www.getambassador.io/yaml/aes-crds.yaml</span>
                    <span className="token plain"> </span>
                    <span className="token operator">&&</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">\</span><br/>
                    <span className="token-plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="QS-token-function">wait</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">--for</span>
                    <span className="token plain"> </span>
                    <span className="token variable">condition</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">established</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">--timeout</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">90s</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">crd -lproduct</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">aes</span>
                    <span className="token plain"> </span>
                    <span className="token operator">&&</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">\</span><br/>
                    <span className="token-plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">apply</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-f https://www.getambassador.io/yaml/aes.yaml</span>
                    <span className="token plain"> </span>
                    <span className="token operator">&&</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">\</span><br/>
                    <span className="token-plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-n</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="QS-token-function">wait</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">--for</span>
                    <span className="token plain"> </span>
                    <span className="token variable">condition</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">available</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">--timeout</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">90s</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">deploy</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-lproduct</span>
                    <span className="token operator">=</span>
                    <span className="token-plain">aes</span>
                  </div>
                </div>
              </div>
            <li>Get your cluster's IP address:</li>
              <div className="styles-module--CodeBlock--1UB4s">
                <div className="QS-codeblockInstall">
                <span className="QS-copyButton"><CopyButton content='
                kubectl get -n ambassador service ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}"'>Copy</CopyButton></span>
                  <div className="token-line">
                    <span className="token-plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">get</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-n</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">service</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="token-plain">-o</span>
                    <span className="token plain"> </span>
                    <span className="token string">"go-template</span>
                    <span className="token string">=</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">range</span>
                    <span className="token string"> </span>
                    <span className="token string">.status</span>
                    <span className="token string">.loadBalancer</span>
                    <span className="token string">.ingress</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">or</span>
                    <span className="token string"> </span>
                    <span className="token string">.ip</span>
                    <span className="token string"> </span>
                    <span className="token string">.hostname</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">&#123;</span>
                    <span className="token string">end</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">&#125;</span>
                    <span className="token string">"</span>
                  </div>
                </div>
              </div>
            <li>Visit <code>http://your-IP-address</code> to login to the console.</li>
            </ul>
          </div>
        </div>

        <span className="QS-helm"><img className="QS-osLogo" src="../../images/helm-navy.png"/></span>

        <div className="QS-aside QS-aside2">
          <div className="QS-asideText">
            <ul id="QS-asideBullets">
              <li>Prefer Helm?  Add this repo to your helm client:</li>
                <div className="styles-module--CodeBlock--1UB4s">
                  <div className="QS-codeblockInstall">
                  <span className="QS-copyButton"><CopyButton content="
                  helm repo add datawire https://www.getambassador.io">Copy</CopyButton></span>
                    <div className="token-line">
                      <span className="token-plain">helm</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">repo</span>
                      <span className="token plain"> </span>
                      <span className="QS-token-function">add</span>
                      <span className="token plain"> </span>
                      <span className="token plain">datawire</span>
                      <span className="token plain"> </span>
                      <span className="QS-token-plain">https://www.getambassador.io</span>
                    </div>
                  </div>
                </div>
              <li>Create the ambassador namespace:</li>
                <div className="styles-module--CodeBlock--1UB4s">
                  <div className="QS-codeblockInstall">
                  <span className="QS-copyButton"><CopyButton content="
                  kubectl create namespace ambassador">Copy</CopyButton></span>
                    <div className="token-line">
                      <span className="token-plain">kubectl</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">create</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">namespace</span>
                      <span className="token plain"> </span>
                      <span className="token plain">ambassador</span>
                    </div>
                  </div>
                </div>

              <li>Install the Ambassador Edge Stack Chart:</li>
              <details open>
                <summary id="helmVersions">&nbsp;Helm3 Users
                </summary>
                  <div id="QS-helm3" className="styles-module--CodeBlock--1UB4s">
                    <div className="QS-codeblockInstall">
                    <span className="QS-copyButton"><CopyButton content="
                    helm install ambassador --namespace ambassador datawire/ambassador">Copy</CopyButton></span>
                      <div className="token-line">
                        <span className="token-plain">helm</span>
                        <span className="token plain"> </span>
                        <span className="token-plain">install</span>
                        <span className="token plain"> </span>
                        <span className="token-plain">ambassador</span>
                        <span className="token plain"> </span>
                        <span className="token plain">--namespace</span>
                        <span className="token plain"> </span>
                        <span className="token-plain">ambassador</span>
                        <span className="token plain"> </span>
                        <span className="token-plain">datawire/ambassador</span>
                      </div>
                    </div>
                  </div>
              </details>
            
              <details>
                <summary id="helmVersions">&nbsp;Helm2 Users
                </summary>
                <div id="QS-helm2" className="styles-module--CodeBlock--1UB4s">
                  <div className="QS-codeblockInstall">
                  <span className="QS-copyButton"><CopyButton content="
                  helm install --name ambassador --namespace ambassador datawire/ambassador">Copy</CopyButton></span>
                    <div className="token-line">
                      <span className="token-plain">helm</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">install</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">--name</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">ambassador</span>
                      <span className="token plain"> </span>
                      <span className="token plain">--namespace</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">ambassador</span>
                      <span className="token plain"> </span>
                      <span className="token-plain">datawire/ambassador</span>
                    </div>
                  </div>
                </div>
              </details>
            
              <li>Install Ambassador Edge Stack:</li>
                <div className="styles-module--CodeBlock--1UB4s">
                    <div className="QS-codeblockInstall">
                    <div className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></div>
                      <div className="token-line">
                        <span className="QS-token-function">edgectl</span>
                        <span className="token plain"> </span>
                        <span className="QS-token-function">install</span>
                      </div>
                    </div>
                </div>
            </ul>
        </div>
      </div>

        <div className="QS-os">
          <span id="QS-showMac" data-os="mac" className="QS-OStype">Mac<img className="QS-osLogo" src="../../images/apple.png" /></span>

          <span id="QS-showLinux" data-os="linux" className="QS-OStype">Linux<img className="QS-osLogo" src="../../images/linux.png" /></span>

          <span id="QS-showWindows" data-os="windows" className="QS-OStype">Windows<img className="QS-osLogo" src="../../images/windows.png" /></span>
        </div>

      <div className="QS-aside QS-aside3">
        <ul id="QS-asideBullets">
          <div id="QS-showMacAside3" data-os="mac" className="QS-asideText">
        
          <li>New user? Get Edgectl, the Ambassador CLI.</li>
          <div className="styles-module--CodeBlock--1UB4s">
            <div className="QS-codeblockInstall">
            <span className="QS-copyButton"><CopyButton content="sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl">Copy</CopyButton></span>
              <div className="token-line">
                <span className="QS-token-function">sudo</span>
                <span className="token plain"> </span>
                <span className="QS-token-function">curl</span>
                <span className="token plain"> -fL https://metriton.datawire.io/downloads/darwin/edgectl <br/>   -o /usr/local/bin/edgectl </span>
                <span className="token operator">&&</span><br/>
                <span className="token plain">  </span>
                <span className="QS-token-function">sudo</span>
                <span className="token plain"> </span>
                <span className="QS-token-function">chmod</span>
                <span className="token plain"> a+x /usr/local/bin/edgectl</span>
              </div>
            </div>
          </div>
        </div>
       

        <div id="QS-showLinuxAside3" data-os="linux" className="
        QS-asideText">
          <li>New user? Get Edgectl, the Ambassador CLI.</li>
          <div className="styles-module--CodeBlock--1UB4s">
            <div className="QS-codeblockInstall">
              <span className="QS-copyButton"><CopyButton content="sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl">Copy</CopyButton></span>
              <div className="token-line">
                <span className="QS-token-function">sudo</span>
                <span className="token plain"> </span>
                <span className="QS-token-function">curl</span>
                <span className="token plain"> -fL https://metriton.datawire.io/downloads/linux/edgectl <br/>   -o /usr/local/bin/edgectl </span>
                <span className="token operator">&&</span><br/>
                <span className="token plain">  </span>
                <span className="QS-token-function">sudo</span>
                <span className="token plain"> </span>
                <span className="QS-token-function">chmod</span>
                <span className="token plain"> a+x /usr/local/bin/edgectl</span>
              </div>
            </div>
          </div>
        </div>

        <div id="QS-showWindowsAside3" data-os="windows" className="QS-asideText">
          <li>New user? Get Edgectl, the Ambassador CLI.</li>
          <code><font size="+1">edgectl.exe</font></code>.
          <div className="styles-module--CodeBlock--1UB4s">
              <div className="QS-codeblockInstall">
                <button><a className="windowsDownloadButton" href="https://metriton.datawire.io/downloads/windows/edgectl.exe" rel="nofollow noopener noreferrer">Download</a></button><span>&nbsp;</span>
                <div className="token-line">
                  <span className="token function"></span>
                </div>
              </div>
          </div>
        </div>


        <div className="QS-asideText">
          <li>Install Ambassador Edge Stack.</li>
            <div className="styles-module--CodeBlock--1UB4s">
              <div className="QS-codeblockInstall">
              <div className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></div>
                <div className="token-line">
                  <span className="QS-token-function">edgectl</span>
                  <span className="token plain"> </span>
                  <span className="QS-token-function">install</span>
                </div>
              </div>
            </div>
        </div>
        </ul>
      </div>

      <div id="QS-fullManual">
        <a href="../../topics/install/">See full-detailed instructions and other install options</a>
      </div>
      
      <div className="QS-main">

        <h2>Ambassador Edge Stack gives you:</h2>
          <div id="QS-mainTextSmall">
            <ul>
              <li className="QS-mainBullet" id="QS-bullet1">First-in-class Kubernetes ingress support with CRD- based configuration</li>

              <li className="QS-mainBullet" id="QS-bullet2">Authentication with OAuth/OIDC integration</li>

              <li className="QS-mainBullet" id="QS-bullet3">Integrations with tools like Grafana, Prometheus, Okta, Consul, and Istio</li>

              <li className="QS-mainBullet" id="QS-bullet4">Layer 7 Load Balancing including support for circuit breakers and automatic retries</li>

              <li className="QS-mainBullet" id="QS-bullet5">A Developer Portal with a fully customizable API catalog plus Swagger/OpenAPI support and more...</li>
            </ul>
          </div>
      </div>
    </div>  
    )
  }
}

export default GettingStarted
