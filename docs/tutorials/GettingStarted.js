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
          document.getElementById("QS-showMacAside1").style.display = "inline-block";
          break;
        case "linux":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside1").style.display = "inline-block";
          break;
        case "windows":
          document.getElementById("QS-showWindows").style.display = "inline-block";
          document.getElementById("QS-showWindowsAside1").style.display = "inline-block";
          break;        
        case "other":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside1").style.display = "inline-block";
      }
    };
    renderHeader(os);
  }

  render() {
    return (
    <div className="QS-grid">

    <div className="QS-os">	
          <span id="QS-showMac" data-os="mac"><img className="QS-osLogo" src="../../images/apple.png" /></span>	

          <span id="QS-showLinux" data-os="linux"><img className="QS-osLogo" src="../../images/linux.png" /></span>	

          <span id="QS-showWindows" data-os="windows"><img className="QS-osLogo" src="../../images/windows.png" /></span>	
        </div>	

      <div className="QS-aside QS-aside1">	
        <ul id="QS-asideBullets">	
          <div id="QS-showMacAside1" data-os="mac" className="QS-asideText">	

          <li>New user? Get Edgectl, the Ambassador CLI</li>	
          <div className="styles-module--CodeBlock--1UB4s">	
            <div className="QS-codeblockInstall">	
            <span className="QS-copyButton"><CopyButton content="sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl">Copy</CopyButton></span>	
              <div className="token-line">	
                <span className="token plain">sudo</span>	
                <span className="token plain"> </span>	
                <span className="token plain">curl</span>	
                <span className="token plain"> -fL https://metriton.datawire.io/downloads/darwin/edgectl <br/>   -o /usr/local/bin/edgectl </span>	
                <span className="token plain">&&</span><br/>	
                <span className="token plain">  </span>	
                <span className="token plain">sudo</span>	
                <span className="token plain"> </span>	
                <span className="token plain">chmod</span>	
                <span className="token plain"> a+x /usr/local/bin/edgectl</span>	
              </div>	
            </div>	
          </div>	
        </div>	

        <div id="QS-showLinuxAside1" data-os="linux" className="	
        QS-asideText">	
          <li>New user? Get Edgectl, the Ambassador CLI</li>	
          <div className="styles-module--CodeBlock--1UB4s">	
            <div className="QS-codeblockInstall">	
              <span className="QS-copyButton"><CopyButton content="sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl">Copy</CopyButton></span>	
              <div className="token-line">	
                <span className="token plain">sudo</span>	
                <span className="token plain"> </span>	
                <span className="token plain">curl</span>	
                <span className="token plain"> -fL https://metriton.datawire.io/downloads/linux/edgectl <br/>   -o /usr/local/bin/edgectl </span>	
                <span className="token plain">&&</span><br/>	
                <span className="token plain">  </span>	
                <span className="token plain">sudo</span>	
                <span className="token plain"> </span>	
                <span className="token plain">chmod</span>	
                <span className="token plain"> a+x /usr/local/bin/edgectl</span>	
              </div>	
            </div>	
          </div>	
        </div>	

        <div id="QS-showWindowsAside1" data-os="windows" className="QS-asideText">	
          <li>New user? Get Edgectl, the Ambassador CLI</li>	
          <div className="styles-module--CodeBlock--1UB4s">	
              <div className="QS-codeblockInstall">	
                <button><a className="windowsDownloadButton" href="https://metriton.datawire.io/downloads/windows/edgectl.exe" rel="nofollow noopener noreferrer">Download edgectl.exe</a></button><span>&nbsp;</span>	
                <div className="token-line">	
                  <span className="token function"></span>	
                </div>	
              </div>	
          </div>	
        </div>	


        <div className="QS-asideText">	
          <li className="QS-asideNoBullets">Install Ambassador Edge Stack</li>	
            <div className="styles-module--CodeBlock--1UB4s">	
              <div className="QS-codeblockInstall">	
              <div className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></div>	
                <div className="token-line">	
                  <span className="token plain">edgectl</span>	
                  <span className="token plain"> </span>	
                  <span className="token plain">install</span>	
                </div>	
              </div>	
            </div>	
        </div>	
        </ul>	
      </div>	

      <span className="QS-k8"><img className="QS-osLogo" src="../../images/kubernetes.png"/></span>

      <div className="QS-aside QS-aside2">
        <div className="QS-asideText">
          <ul id="QS-asideBullets">
            <li>Have Kubernetes? Deploy with YAML</li>
              <div className="styles-module--CodeBlock--1UB4s">
                <div className="QS-codeblockInstall">
                <span className="QS-copyButton"><CopyButton content="
                kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml && kubectl wait --for condition=established --timeout=90s crd -lproduct=aes && kubectl apply -f https://www.getambassador.io/yaml/aes.yaml && kubectl -n ambassador wait --for condition=available --timeout=90s deploy -lproduct=aes">Copy</CopyButton></span>
                  <div className="token-line">
                    <span className="token plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token plain">apply</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-f https://www.getambassador.io/yaml/aes-crds.yaml</span>
                    <span className="token plain"> </span>
                    <span className="token plain">&&</span>
                    <span className="token plain"> </span>
                    <span className="token plain">\</span><br/>
                    <span className="token plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token plain">wait</span>
                    <span className="token plain"> </span>
                    <span className="token plain">--for</span>
                    <span className="token plain"> </span>
                    <span className="token plain">condition</span>
                    <span className="token plain">=</span>
                    <span className="token plain">established</span>
                    <span className="token plain"> </span>
                    <span className="token plain">--timeout</span>
                    <span className="token plain">=</span>
                    <span className="token plain">90s</span>
                    <span className="token plain"> </span>
                    <span className="token plain">crd -lproduct</span>
                    <span className="token plain">=</span>
                    <span className="token plain">aes</span>
                    <span className="token plain"> </span>
                    <span className="token plain">&&</span>
                    <span className="token plain"> </span>
                    <span className="token plain">\</span><br/>
                    <span className="token plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token plain">apply</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-f https://www.getambassador.io/yaml/aes.yaml</span>
                    <span className="token plain"> </span>
                    <span className="token plain">&&</span>
                    <span className="token plain"> </span>
                    <span className="token plain">\</span><br/>
                    <span className="token plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-n</span>
                    <span className="token plain"> </span>
                    <span className="token plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="token plain">wait</span>
                    <span className="token plain"> </span>
                    <span className="token plain">--for</span>
                    <span className="token plain"> </span>
                    <span className="token plain">condition</span>
                    <span className="token plain">=</span>
                    <span className="token plain">available</span>
                    <span className="token plain"> </span>
                    <span className="token plain">--timeout</span>
                    <span className="token plain">=</span>
                    <span className="token plain">90s</span>
                    <span className="token plain"> </span>
                    <span className="token plain">deploy</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-lproduct</span>
                    <span className="token plain">=</span>
                    <span className="token plain">aes</span>
                  </div>
                </div>
              </div>
            <li className="QS-asideNoBullets">Get your cluster's IP address</li>
              <div className="styles-module--CodeBlock--1UB4s">
                <div className="QS-codeblockInstall">
                <span className="QS-copyButton"><CopyButton content='
                kubectl get -n ambassador service ambassador -o "go-template={{range .status.loadBalancer.ingress}}{{or .ip .hostname}}{{end}}"'>Copy</CopyButton></span>
                  <div className="token-line">
                    <span className="token plain">kubectl</span>
                    <span className="token plain"> </span>
                    <span className="token plain">get</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-n</span>
                    <span className="token plain"> </span>
                    <span className="token plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="token plain">service</span>
                    <span className="token plain"> </span>
                    <span className="token plain">ambassador</span>
                    <span className="token plain"> </span>
                    <span className="token plain">-o</span>
                    <span className="token plain"> </span>
                    <span className="token plain">"go-template</span>
                    <span className="token plain">=</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">range</span>
                    <span className="token plain"> </span>
                    <span className="token plain">.status</span>
                    <span className="token plain">.loadBalancer</span>
                    <span className="token plain">.ingress</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">or</span>
                    <span className="token plain"> </span>
                    <span className="token plain">.ip</span>
                    <span className="token plain"> </span>
                    <span className="token plain">.hostname</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">&#123;</span>
                    <span className="token plain">end</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">&#125;</span>
                    <span className="token plain">"</span>
                  </div>
                </div>
              </div>
            <li className="QS-asideNoBullets">Visit <code id="k8Code">http://your-IP-address</code> to login to the console.</li>
            </ul>
          </div>
        </div>

        <span className="QS-helm"><img className="QS-osLogo" src="../../images/helm-navy.png"/></span>

        <div className="QS-aside QS-aside3">
          <div className="QS-asideText">
            <ul id="QS-asideBullets">
              <li>Prefer Helm?  Add this repo to your Helm client</li>
                <div className="styles-module--CodeBlock--1UB4s">
                  <div className="QS-codeblockInstall">
                  <span className="QS-copyButton"><CopyButton content="
                  helm repo add datawire https://www.getambassador.io">Copy</CopyButton></span>
                    <div className="token-line">
                      <span className="token plain">helm</span>
                      <span className="token plain"> </span>
                      <span className="token plain">repo</span>
                      <span className="token plain"> </span>
                      <span className="token plain">add</span>
                      <span className="token plain"> </span>
                      <span className="token plain">datawire</span>
                      <span className="token plain"> </span>
                      <span className="token plain">https://www.getambassador.io</span>
                    </div>
                  </div>
                </div>

              <li className="QS-asideNoBullets">Install the Ambassador Edge Stack chart</li>
                <div id="helmVersionWrapper">

                  <div id="helm2Block">
                    <details open>
                      <summary id="helmVersions">&nbsp;Helm2
                      </summary>
                      <div id="QS-helm2" className="styles-module--CodeBlock--1UB4s">
                        <div className="QS-codeblockInstall">
                        <span className="QS-copyButton"><CopyButton content="
                        kubectl create namespace ambassador && helm install --name ambassador --namespace ambassador datawire/ambassador">Copy</CopyButton></span>
                          <div className="token-line">
                            <span className="token plain">kubectl</span>
                            <span className="token plain"> </span>
                            <span className="token plain">create</span>
                            <span className="token plain"> </span>
                            <span className="token plain">namespace</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">&&</span>
                            <span className="token plain"> </span>
                            <span className="token plain">\</span><br/>
                            <span className="token plain">helm</span>
                            <span className="token plain"> </span>
                            <span className="token plain">install</span>
                            <span className="token plain"> </span>
                            <span className="token plain">--name</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">--namespace</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">datawire/ambassador</span>
                          </div>
                        </div>
                      </div>
                    </details>
                  </div>

                  <div id="helm3Block">
                    <details open>
                      <summary id="helmVersions">&nbsp;Helm3
                      </summary>
                      <div id="QS-helm3" className="styles-module--CodeBlock--1UB4s">
                        <div className="QS-codeblockInstall">
                        <span className="QS-copyButton"><CopyButton content="
                        kubectl create namespace ambassador && helm install ambassador --namespace ambassador datawire/ambassador">Copy</CopyButton></span>
                            <div className="token-line">
                            <span className="token plain">kubectl</span>
                            <span className="token plain"> </span>
                            <span className="token plain">create</span>
                            <span className="token plain"> </span>
                            <span className="token plain">namespace</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">&&</span>
                            <span className="token plain"> </span>
                            <span className="token plain">\</span><br/>
                            <span className="token plain">helm</span>
                            <span className="token plain"> </span>
                            <span className="token plain">install</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">--namespace</span>
                            <span className="token plain"> </span>
                            <span className="token plain">ambassador</span>
                            <span className="token plain"> </span>
                            <span className="token plain">datawire/ambassador</span>
                          </div>
                        </div>
                      </div>
                    </details>
                  </div>
                </div>
            
              <li className="QS-asideNoBullets">Install Ambassador Edge Stack</li>
                <div className="styles-module--CodeBlock--1UB4s">
                    <div className="QS-codeblockInstall">
                    <div className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></div>
                      <div className="token-line">
                        <span className="token plain">edgectl</span>
                        <span className="token plain"> </span>
                        <span className="token plain">install</span>
                      </div>
                    </div>
                </div>
            </ul>
        </div>
      </div>

      <div id="QS-blank"></div>

      <div id="QS-fullManual">
        <a href="../../topics/install/">See full-detailed instructions and other install options</a>
      </div>

      <div className="QS-blackbird-image">
        <img alt="Ambassador's OpenSource Blackbird" src="/images/features-page-bird.svg"/>
      </div>

      <div className="QS-Spin">
       <div className="QS-asideText">
         Take it for a spin!<br/>
          <span className="QS-spinText">➞ <a href="../../tutorials/quickstart-demo/">See how Ambassador works with a service</a></span><br/>
          <span id="QS-customLink" className="QS-spinText">➞ <a href="../../topics/using/">Check out custom options and integrations</a></span><br/>
        </div>
       </div>
      
      <div className="QS-main">

        <h2>Ambassador Edge Stack gives you</h2>
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
