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
      redisInUse: { type: Boolean }
    };
  }

  constructor() {
    super();

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
      
      .col2 a.cta  {
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
      
      .col2 a.cta  {
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
      
      .header_con .col2 a.cta  {
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
      
    `;
  }

  render() {
    return html`
      <div class="header_con">
        <div class="col">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 23.78 23.37"><defs><style>.cls-1{fill:#fff;}</style></defs><title>debug</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M23.16,13.94H20a9.24,9.24,0,0,0-1.73-4.85l2.29-2.28a.62.62,0,0,0-.88-.88L17.43,8.14a8.86,8.86,0,0,0-1.09-.94,3.35,3.35,0,0,0,.1-.87,4.57,4.57,0,0,0-2.37-4,4,4,0,0,1,1.68-1.12.62.62,0,0,0,.4-.78.61.61,0,0,0-.78-.4,5.27,5.27,0,0,0-2.53,1.85,4.56,4.56,0,0,0-1.9,0A5.31,5.31,0,0,0,8.41,0a.62.62,0,0,0-.78.4A.62.62,0,0,0,8,1.21,3.92,3.92,0,0,1,9.71,2.33a4.56,4.56,0,0,0-2.38,4,3.34,3.34,0,0,0,.11.87,8.18,8.18,0,0,0-1.09.94L4.14,5.93a.62.62,0,1,0-.88.88L5.55,9.09a9.33,9.33,0,0,0-1.74,4.85H.62a.62.62,0,0,0,0,1.24h3.2A9.1,9.1,0,0,0,5.55,20L3.26,22.31a.62.62,0,0,0,0,.87.6.6,0,0,0,.88,0L6.35,21a7.75,7.75,0,0,0,5.54,2.4A7.75,7.75,0,0,0,17.43,21l2.21,2.21a.63.63,0,0,0,.44.19.65.65,0,0,0,.44-.19.62.62,0,0,0,0-.87L18.23,20A9.1,9.1,0,0,0,20,15.18h3.2a.62.62,0,0,0,0-1.24ZM11.89,3a3.32,3.32,0,0,1,3.32,3.31,1.37,1.37,0,0,1-.3,1c-.42.42-1.5.41-2.63.41H11.5c-1.14,0-2.21,0-2.63-.41a1.37,1.37,0,0,1-.3-1A3.32,3.32,0,0,1,11.89,3ZM5,14.56a7.83,7.83,0,0,1,3-6.29C8.82,9,10,9,11.27,9V22.09C7.78,21.75,5,18.5,5,14.56Zm7.48,7.53V9h.13a4.51,4.51,0,0,0,3.07-.71,7.85,7.85,0,0,1,3,6.29C18.75,18.5,16,21.75,12.51,22.09Z"/></g></g></svg>
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
          </dl>
  
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

  setLogLevel(level) {
    let formdata = new FormData();
    formdata.append('loglevel', level);

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
