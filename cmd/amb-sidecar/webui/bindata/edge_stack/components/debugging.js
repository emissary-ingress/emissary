import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import { registerContextChangeHandler, useContext } from '/edge_stack/components/context.js'
import { getCookie } from '/edge_stack/components/cookies.js';

export class Debugging extends LitElement {
  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      diagd: {type: Object},
    };
  }

  constructor() {
    super();

    const [currentDiagd, setDiagd] = useContext('aes-api-diag', null);
    this.onDiagdChange(currentDiagd);
    registerContextChangeHandler('aes-api-diag', this.onDiagdChange.bind(this));
  }

  static get styles() {
    return css`
      dl {
        display: grid;
        grid-template-columns: max-content;
        grid-gap: 0 1em;
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
    `;
  }

  render() {
    return html`
      <fieldset>
        <legend>System info</legend>
        <dl>

          <dt>Ambassador version</dt>
          <dd>${this.diagd.system.version}</dd>

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

          <dt>KNative support</dt>
          <dd>${this.diagd.system.knative_enabled ? "enabled" : "disabled"}</dd>

          <dt>StatsD support</dt>
          <dd>${this.diagd.system.statsd_enabled ? "enabled" : "disabled"}</dd>

          <dt>Endpoint routing</dt>
          <dd>${this.diagd.system.statsd_enabled ? "enabled" : "disabled"}</dd>

        </dl>
      </fieldset>

      <fieldset>
        <legend>Log level</legend>

        <dl>
          <dt>Current log level</dt>
          <dd>${this.diagd.loginfo.all}</dd>
        </dl>

        <div>
          <button @click=${()=>{this.setLogLevel('debug')}}>Set log level to <q>debug</q></button>
          <button @click=${()=>{this.setLogLevel('info')}}>Set log level to <q>info</q></button>
        </div>

      </fieldset>

      <fieldset>
        <legend>Ambassador configuration ${
           this.diagd.system.env_good
             ? html`<span style="color: green">looks good</span>`
             : html`<span style="color: red; font-weight: bold">has issues</span>`
        }</legend>

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

      </fieldset>

      ${this.diagd.errors.length === 0 ? html`` : html`
      <fieldset>
        <legend>CONFIGURATION ERRORS</legend>

        <ul>${this.diagd.errors.sort().map(([error_target, error_message]) => html`
          <li>
            ${error_target ? html`<span class="error_target">${error_target}</span>:` : html``}
            <span class="error_message">${error_message}</span>
          </li>
        `)}</ul>

      </fieldset>
      `}
    `;
  }

  // internal ////////////////////////////////////////////////////////

  onDiagdChange(snapshot) {
    this.diagd = (('system' in (snapshot||{})) ? snapshot :
     {
       system: {
         env_status: {},
       },
       envoy_status: {},
       loginfo: {},
       errors: [],
     });
  }

  setLogLevel(level) {
    let formdata = new FormData();
    formdata.append('loglevel', level);

    fetch('/edge_stack/api/log-level', {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth"),
      },
      body: formdata,
    });
  }
}

customElements.define('dw-debugging', Debugging);
