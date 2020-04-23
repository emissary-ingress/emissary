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
          document.getElementById("QS-showMacAside2").style.display = "inline-block";
          break;
        case "linux":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside1").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside2").style.display = "inline-block";
          break;
        case "windows":
          document.getElementById("QS-showWindows").style.display = "inline-block";
          document.getElementById("QS-showWindowsAside1").style.display = "inline-block";
          document.getElementById("QS-showWindowsAside2").style.display = "inline-block";
          break;        
        case "other":
          document.getElementById("QS-showLinux").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside1").style.display = "inline-block";
          document.getElementById("QS-showLinuxAside2").style.display = "inline-block";
      }
    };
    renderHeader(os);
  }

  render() {
    return (
    <div className="QS-grid">
      <span className="QS-rightTopWrapper">
        <div id="QS-header1">
          <span id="QS-showMac" data-os="mac" className="QS-header1">Mac<img className="os-logo" src="../../images/apple.png" /></span>

          <span id="QS-showLinux" data-os="linux" className="QS-header1">Linux<img className="os-logo" src="../../images/linux.png" /></span>

          <span id="QS-showWindows" data-os="windows" className="QS-header1">Windows<img className="os-logo" src="../../images/windows.png" /></span>
        </div>

        <div className="QS-moreInstallOptions"><a id="QS-moreInstallOptions"    href="../../topics/install/">(more install options)</a>
        </div>
      </span>
    

      <div className="QS-aside QS-aside1">
        <div id="QS-showMacAside1" data-os="mac" className="QS-asideText">
          1. Get Edgectl, the Ambassador installer.
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

        <div id="QS-showLinuxAside1" data-os="linux" className="
        QS-asideText">
          1. Get Edgectl, the Ambassador installer.
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

        <div id="QS-showWindowsAside1" data-os="windows" className="QS-asideText">
          1. Download installer <code><font size="+1">edgectl.exe</font></code>.
          <div className="styles-module--CodeBlock--1UB4s">
              <div className="QS-codeblockInstall">
                <button><a className="windowsDownloadButton" href="https://metriton.datawire.io/downloads/windows/edgectl.exe" rel="nofollow noopener noreferrer">Download</a></button><span>&nbsp;</span>
                <div className="token-line">
                  <span className="token function"></span>
                </div>
              </div>
          </div>
        </div>

      </div>

      <div className="QS-aside QS-aside2">
        <div id="QS-showMacAside2" data-os="mac" className="QS-asideText">
          2. Install Ambassador.
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

        <div id="QS-showLinuxAside2" data-os="linux" className="QS-asideText">
          2. Install Ambassador.
          <div className="styles-module--CodeBlock--1UB4s">
            <div className="QS-codeblockInstall">
              <span className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></span>
              <div className="token-line">
                <span className="QS-token-function">edgectl</span>
                <span className="token plain"> </span>
                <span className="QS-token-function">install</span>
              </div>
            </div>
          </div>
        </div>

        <div id="QS-showWindowsAside2" data-os="windows" className="QS-asideText">2. Install Ambassador.
          <div className="styles-module--CodeBlock--1UB4s">
              <div className="QS-codeblockInstall">
                <span className="QS-copyButton"><CopyButton content="edgectl install">Copy</CopyButton></span>
                <div className="token-line">
                  <span className="QS-token-function">edgectl</span>
                  <span className="token plain"> </span>
                  <span className="QS-token-function">install</span>
                </div>
              </div>
            </div>
        </div>
      </div>


      <div className="QS-aside QS-aside3">
        <div className="QS-asideText">
          3. Take it for a spin! 
          
              <div className="QS-codeblockInstall QS-Aside3codeblockInstall">
                <span className="QS-aside3prose">➞ <a href="../../tutorials/quickstart-demo/">See how Ambassador works with a service</a>.</span><br/>
                <span className="QS-aside3prose">➞ <a href="../../topics/using/">Check out custom options and integrations</a>.</span>
              </div>
              <div className="QS-blackbird-image">
                <img alt="Ambassador's OpenSource Blackbird" src="/images/features-page-bird.svg"/>
              </div>
        </div>
      </div>

      <div id="QS-fullManual">
        <a href="../../topics/install/">Want to view the full installation manual?</a>
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
