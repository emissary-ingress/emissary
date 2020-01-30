import { LitElement, html, css } from '../vendor/lit-element.min.js'
import { Snapshot } from './snapshot.js'
import { getCookie } from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

export class Debugging extends LitElement {
  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      diagd: {type: Object},
      licenseClaims: { type: Object },
      featuresOverLimit: { type: Object },
      redisInUse: { type: Boolean },

    };
  }

  constructor() {
    super();

    const logOptions = [
      {value: "debug", label: "DEBUG"},
      {value: "info", label: "INFO"},
      {value: "trace", label: "TRACE"}
    ];

    this.logOptions = logOptions;
    

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
  }

  static get styles() {
    return css`
      * {
        margin: 0;
        padding: 0;
        border: 0;
        position: relative;
        box-sizing: border-box
      }
      
      *, textarea {
        vertical-align: top
      }
      
      .card {
        background: #fff;
        border-radius: 10px;
        padding: 10px 30px 10px 30px;
        box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
        width: 100%;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row;
        -webkit-flex: 1 1 1;
        -ms-flex: 1 1 1;
        flex: 1 1 1;
        margin: 30px 0 0;
        font-size: .9rem;
      }
      
      .card, .card .col .con {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex
      }

      .card .col .con {
        margin: 10px 0;
        -webkit-flex: 1;
        -ms-flex: 1;
        flex: 1;
        -webkit-justify-content: flex-end;
        -ms-flex-pack: end;
        justify-content: flex-end;
        height: 30px
      }
      
      .card .col, .card .col .con label, .card .col2, .col2 a.cta .label {
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        -ms-grid-row-align: center;
        align-self: center
      }
      
      .col2 a.cta {
        text-decoration: none;
        border: 2px #efefef solid;
        border-radius: 10px;
        width: 90px;
        padding: 6px 8px;
        -webkit-flex: auto;
        -ms-flex: auto;
        flex: auto;
        margin: 10px auto;
        color: #000;
        transition: all .2s ease;
        cursor: pointer;
      }
      
      .col2 a.cta .label {
        text-transform: uppercase;
        font-size: .8rem;
        font-weight: 600;
        line-height: 1rem;
        padding: 0 0 0 10px;
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto
      }
      
      .col2 a.cta svg {
        width: 15px;
        height: auto
      }
      
      .col2 a.cta svg path, .col2 a.cta svg polygon {
        transition: fill .7s ease;
        fill: #000
      }
      
      .col2 a.cta:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid
      }
      
      .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
        transition: fill .2s ease;
        fill: #5f3eff
      }
      
      .col2 a.cta {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .col2 a.off {
        display: none;
      }

      ul {
        padding-left: 2em;
      }
      dl {
        display: grid;
        grid-template-columns: max-content;
        grid-gap: 0 1em;
        margin: 0.5em 0 0.5em 0;
      }
      dl > dt {
        grid-column: 1 / 2;
      }
      dl > dt::after {
        content: ":";
      }
      dl > dd {
        grid-column: 2 / 3;
      }

      .error_target: {
        font-family: monospace;
      }
      .error_message {
        color: red;
      }
      div.over_limit {
        color: red;
        font-weight: bold;
      }
      div.over_limit a {
        color: red;
      }

      * {
        margin: 0;
        padding: 0;
        border: 0;
        position: relative;
        box-sizing: border-box
      }
      
      *, textarea {
        vertical-align: top
      }
      
      
      .header_con, .header_con .col {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }
      
      .header_con {
        margin: 30px 0 0;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .header_con .col {
        -webkit-flex: 0 0 80px;
        -ms-flex: 0 0 80px;
        flex: 0 0 80px;
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center;
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        align-self: center;
        -webkit-flex-direction: column;
        -ms-flex-direction: column;
        flex-direction: column
      }
      
      .header_con .col svg {
        width: 100%;
        height: 60px
      }
      
      .header_con .col svg path {
        fill: #5f3eff
      }

      .header_con .col img {
        width: 100%;
        height: 60px
      }
      
      .header_con .col img path {
        fill: #5f3eff
      }
      
      .header_con .col:nth-child(2) {
        -webkit-flex: 2 0 auto;
        -ms-flex: 2 0 auto;
        flex: 2 0 auto;
        padding-left: 20px
      }
      
      .header_con .col h1 {
        padding: 0;
        margin: 0;
        font-weight: 400
      }
      
      .header_con .col p {
        margin: 0;
        padding: 0
      }
      
      .header_con .col2, .col2 a.cta .label {
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        -ms-grid-row-align: center;
        align-self: center
      }
      
      .col2 a.cta {
        text-decoration: none;
        border: 2px #efefef solid;
        border-radius: 10px;
        width: 90px;
        padding: 6px 8px;
        -webkit-flex: auto;
        -ms-flex: auto;
        flex: auto;
        margin: 10px auto;
        color: #000;
        transition: all .2s ease;
        cursor: pointer;
      }
      
      .header_con .col2 a.cta {
        border-color: #c8c8c8;
      }
      
      .col2 a.cta .label {
        text-transform: uppercase;
        font-size: .8rem;
        font-weight: 600;
        line-height: 1rem;
        padding: 0 0 0 10px;
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto
      }
      
      .col2 a.cta svg {
        width: 15px;
        height: auto
      }
      
      .col2 a.cta svg path, .col2 a.cta svg polygon {
        transition: fill .7s ease;
        fill: #000
      }
      
      .col2 a.cta:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid
      }
      
      .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
        transition: fill .2s ease;
        fill: #5f3eff
      }
      
      .col2 a.cta {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .col2 a.off {
        display: none;
      }

      .logo {
        filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
      }
      span.logLabel {
        vertical-align: center;
        top: 3px;
        bottom: 0;
        right: 0;
      }
      div.logDiv {
        position: relative;
        margin-top: 10px;
      }
      select.logSelector {
        height: 25px;
        width: 100px;
        border-radius: 0;
        padding-left: 15px;
        margin-left: 5px;
        border: 2px #efefef solid;
        border-radius: 10px;
        font-size: .8rem;
        font-weight: 600;

        /* Removes the default <select> styling */
        -webkit-appearance: none;
        -moz-appearance: none;
        appearance: none;

        /* Positions background arrow image */
        background-image: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAh0lEQVQ4T93TMQrCUAzG8V9x8QziiYSuXdzFC7h4AcELOPQAdXYovZCHEATlgQV5GFTe1ozJlz/kS1IpjKqw3wQBVyy++JI0y1GTe7DCBbMAckeNIQKk/BanALBB+16LtnDELoMcsM/BESDlz2heDR3WePwKSLo5eoxz3z6NNcFD+vu3ij14Aqz/DxGbKB7CAAAAAElFTkSuQmCC');
        background-repeat: no-repeat;
        background-position: 75px center;
      }
    `;
  }

  render() {
    return html`
      <div class="header_con">
        <div class="col">
          <img alt="debugging logo" class="logo" src="../images/svgs/debugging.svg">
            <defs><style>.cls-1{fill:#fff;}</style></defs>
              <g id="Layer_2" data-name="Layer 2">
                <g id="Layer_1-2" data-name="Layer 1"></g>
              </g>
          </img>
        </div>
        <div class="col">
          <h1>Debugging</h1>
          <p>System information.</p>
        </div>
      </div>
      <div>
      
      <div class="card">
        <div class="col">
          <h3>System info</h3>
          <dl>
  
            <dt>Ambassador version</dt>
            <dd>${this.diagd.system.version}</dd>
            
            <dt>License</dt>
            <dd>${this.licenseClaims.customer_email || this.licenseClaims.customer_id}</dd>
  
            <dt>Hostname</dt>
            <dd>${this.diagd.system.hostname}</dd>
  
            <dt>Cluster ID</dt>
            <dd>${this.diagd.system.cluster_id}</dd>
  
            <dt>Envoy status</dt>
            <dd>${
             this.diagd.envoy_status.ready
               ? html`ready (last status report ${this.diagd.envoy_status.since_update})`
               : (this.diagd.envoy_status.alive
                   ? html`alive but not yet ready (running ${this.diagd.envoy_status.uptime})`
                   : html`not running`)
            }</dd>
            
            <dt>Redis status</dt>
            <dd>
            ${this.redisInUse
              ? html`configured`
              : html`Authentication and Rate Limiting are disabled as Ambassador Edge Stack is not configured to use Redis. Please follow the <a href="https://www.getambassador.io/user-guide/install">Ambassador Edge Stack installation guide</a> to complete your setup.`}
            </dd>
  
          </dl>
          <dl>
  
            <dt>Ambassador ID</dt>
            <dd>${this.diagd.system.ambassador_id}</dd>
  
            <dt>Ambassador namespace</dt>
            <dd>${this.diagd.system.ambassador_namespace}</dd>
  
            <dt>Ambassador single namespace</dt>
            <dd>${this.diagd.system.single_namespace}</dd>
  
          </dl>
          <dl>
  
            <dt>Knative support</dt>
            <dd>${this.diagd.system.knative_enabled ? "enabled" : "disabled"}</dd>
  
            <dt>StatsD support</dt>
            <dd>${this.diagd.system.statsd_enabled ? "enabled" : "disabled"}</dd>
  
            <dt>Endpoint routing</dt>
            <dd>${this.diagd.system.statsd_enabled ? "enabled" : "disabled"}</dd>
  
          </dl>
          
          ${this.featuresOverLimit.length > 0 
            ? html`<div class="over_limit">You've reached the <a href="https://www.getambassador.io/editions/">usage limits</a> for your license. If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.</div>`
            : html``
          }
        </div>
      </div>

      <div class="card">
        <div class="col2">
          <h3>Log level</h3>
          <dl>
            <dt>Current log level</dt>
            <dd>${this.diagd.loginfo.all}</dd>
            <dd>${this.level}</dd>
          </dl>
          <div class="logDiv"><span class="logLabel">Set Log Level:</span>
            <select class="logSelector" 
            
            @change=${this.onChangeSetLogLevel.bind(this)}>
              <option ?selected=${this.setLogLevel('debug')} value="debug">DEBUG</option>
              <option ?selected=${this.setLogLevel('info')} value="info">INFO</option>
              <option ?selected=${this.setLogLevel==="trace"} value="trace">TRACE</option>
            </select>
          </div>


          <a class="cta" style="width: auto" @click=${()=>{this.setLogLevel('debug')}}><div class="label">Set log level to <q>debug</q></div></a>
          <a class="cta" style="width: auto" @click=${()=>{this.setLogLevel('info')}}><div class="label">Set log level to <q>info</q></div></a>
        </div>
      </div>

      <div class="card">
        <div class="col">
          <h3>Ambassador configuration ${
            this.diagd.system.env_good
              ? html`<span style="color: green">looks good</span>`
              : html`<span style="color: red; font-weight: bold">has issues</span>`
          }</h3>
  
          <ul>${Object.entries(this.diagd.system.env_status).sort().map(([sys_name, sys_stat]) => html`
            <li>
              ${sys_stat.status
                ? html`<span style="color: green">&#x2713</span> ${sys_name} passed`
                : html`<span style="color: red">&#x2717 ${sys_name} failed</span>`
              }
              <ul>${sys_stat.specifics.map(([specific_status, specific_text]) => html`
                <li>
                  ${specific_status
                    ? html`<span style="color: green">&#x2713</span>`
                    : html`<span style="color: red">&#x2717</span>`}
                  ${specific_text}
                </li>
              `)}</ul>
            </li>
          `)}</ul>

        </div>
      </div>

      ${this.diagd.errors.length === 0 ? html`` : html`
      <div class="card">
        <div class="col">
          <h3>Configuration errors</h3>
          <ul>${this.diagd.errors.sort().map(([error_target, error_message]) => html`
            <li>
              ${error_target ? html`<span class="error_target">${error_target}</span>:` : html``}
              <span class="error_message">${error_message}</span>
            </li>
          `)}</ul>
        </div>
      </div>`}
    `;
  }

  // internal ////////////////////////////////////////////////////////

  onSnapshotChange(snapshot) {
    let diagnostics = snapshot.getDiagnostics();
    this.diagd = (('system' in (diagnostics||{})) ? diagnostics :
     {
       system: {
         env_status: {},
       },
       envoy_status: {},
       loginfo: {},
       errors: [],
     });
    this.licenseClaims = snapshot.getLicense().Claims || {};
    this.featuresOverLimit = snapshot.getLicense().FeaturesOverLimit || [];
    this.redisInUse = snapshot.getRedisInUse();
  }

  onChangeSetLogLevel(e) {
    this.level = e.target.options[e.target.selectedIndex].value;
    console.log("this.level in onChangeSetLogLevel is " + this.level);
    this.setLogLevel(this.level);
  }

  setLogLevel(level) {
    console.log("level in setLogLevel is " + level);
    let formdata = new FormData();
    formdata.append('loglevel', level);
    console.log("level in formdata.append is " + level);

    ApiFetch('/edge_stack/api/log-level', {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth"),
      },
      body: formdata,
    });
  }
}

customElements.define('dw-debugging', Debugging);
