/* ===================================================================================*/
/* Dashboard and dashboard element classes using LitElement.                          */
/* ===================================================================================*/

import { LitElement, html, css, svg } from '/edge_stack/vendor/lit-element.min.js';
import { Snapshot } from '/edge_stack/components/snapshot.js';
import { License } from '/edge_stack/components/license.js';

/**
 * This is a Promise-like object used to synchronize between google charts loaded callback and the
 * updateCompleted callback. Specifically, the function that we execute at updateCompleted requires
 * the google charts to have loaded completely, but since that is down asynchronously, we have to
 * wait.
 */
let WhenChartsAreLoadedPromise = {
    _pending: [],
    then: function (f) {
        /* the charts library isn't loaded yet, so queue up the
         * lambdas until the library is loaded */
        this._pending.push(f);
    },
    resolve: function() {
        /* when the charts library is loaded, I replace myself with a
         * new object that doesn't queue the lambdas and then.. */
        WhenChartsAreLoadedPromise = {
            then: function (f) { f(); },
            resolve: function () {}
        };
        /* ..then execute all the pending lambdas */
      for(let i = 0; i < this._pending.length; i++) {
            this._pending[i]();
        }
    }
};

/* ===================================================================================
 * Dashboard panels defined as objects.  Declared here so that they can be added
 * to the Dashboard on instantiation.
 * ===================================================================================
 *
 * The Dashboard panel objects are objects that define three functions:
 *   render() => string
 *        Return a html template for the panel.
 *   onSnapshotChange(snapshot) => (no return value)
 *        Called each time a new snapshot is received from Ambassador. This function
 *        is used to update any internal state for the panel from the data in the snapshot.
 *   draw(shadow_root) => (no return value)
 *        Use the google chart apis to create a chart and attach it to panel's element
 *        in the shadow DOM. This function does nothing if there is no chart in the panel.
 */

/* Rendering utilities */

/* No, One, many things... */
let countString = function(count, singular_text, plural_text) {
    switch(count) {
      case 0:  return `No ${plural_text}`;
      case 1:  return `1 ${singular_text}`;
      default: return `${count} ${plural_text}`;
    }
};

/* Rendering an arc in an SVG div */

let renderArc = function(color, start_rad, end_rad) {
    const cos = Math.cos;
    const sin = Math.sin;
    const π   = Math.PI;

    const f_matrix_times = (( [[a,b], [c,d]], [x,y]) => [ a * x + b * y, c * x + d * y]);
    const f_rotate_matrix = ((x) => [[cos(x),-sin(x)], [sin(x), cos(x)]]);
    const f_vec_add = (([a1, a2], [b1, b2]) => [a1 + b1, a2 + b2]);

    const f_svg_ellipse_arc = (([cx,cy],[rx,ry], [t1, Δ], φ ) => {
      /* [
      returns a SVG path element that represent a ellipse.
      cx,cy → center of ellipse
      rx,ry → major minor radius
      t1 → start angle, in radian.
      Δ → angle to sweep, in radian. positive.
      φ → rotation on the whole, in radian
      url: SVG Circle Arc http://xahlee.info/js/svg_circle_arc.html
      Version 2019-06-19
       ] */
      Δ = Δ % (2*π);
      const rotMatrix = f_rotate_matrix (φ);
      const [sX, sY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1), ry * sin(t1)] ), [cx,cy] ) );
      const [eX, eY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1+Δ), ry * sin(t1+Δ)] ), [cx,cy] ) );
      const fA = ( (  Δ > π ) ? 1 : 0 );
      const fS = ( (  Δ > 0 ) ? 1 : 0 );
      //return "M " + sX + " " + sY + " L " + (sX + 100) + " " + sY;
      return "M " + sX + " " + sY + " A " + [ rx , ry , φ / (2*π) *360, fA, fS, eX, eY ].join(" ")
    });

    var result = svg`
        <g stroke="${color}" fill="none" stroke-linecap="round" stroke-width="8">
            <path d="${f_svg_ellipse_arc([100,100], [90,90], [start_rad,end_rad], 0)}"/>
        </g>
    `;

    return result
  };


let LicensePanel = {
  _title: "License",
  _elementId: "license",
  licenseClaims: {},
  featuresOverLimit: [],

  render: function() {
    return (this.featuresOverLimit.length > 0 || !this.isLicenseRegistered() ? html`
    <div class="element" style="cursor:pointer">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content centered" id=“${this._elementId}”>
        <div>
        ${this.featuresOverLimit.length > 0
          ? html`<div style="color:red; font-weight: bold">
                  You've reached the usage limits for your license.
                  ${this.isLicenseRegistered() 
                    ? html`<br/>If you need to use Ambassador beyond the current limits, <a href="https://www.getambassador.io/contact/">please contact Datawire</a> for an Enterprise license.` 
                    : html``}
                 </div>`
          : html``
        }
        <br/>
        ${!this.isLicenseRegistered() ? html`<dw-signup></dw-signup>` : html``}
        </div>
      </div>
    </div>` : html``)
  },

  onSnapshotChange: function(snapshot) {
    this.licenseClaims = snapshot.getLicense().Claims || {};
    this.featuresOverLimit = snapshot.getLicense().FeaturesOverLimit || [];
	},

  isLicenseRegistered: function() {
    return this.licenseClaims && this.licenseClaims.customer_id !== License._UNREGISTERED_CUSTOMER_ID;
  },

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },
};

/**
 * Panel showing a count of the Hosts, Mappings, and Services
 */
let CountsPanel = {
  _title: "Counts",
  _elementId: "counts",
  _hostsCount: 0,
  _mappingsCount: 0,
  _servicesCount: 0,

  render: function() {
    return html`
    <div class="element" style="cursor:pointer">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content " id=“${this._elementId}”>
        <p style="margin-top: 2.8em"><span class = "status" @click=${this.onClickHosts}>${countString(this._hostsCount, "Host", "Hosts")}</span></p>
        <p><span class = "status" @click=${this.onClickMappings}>${countString(this._mappingsCount, "Mapping", "Mappings")}</span></p>
        <p><span class = "status" @click=${this.onClickServices}>${countString(this._servicesCount, "Service", "Services")}</span></p>
      </div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    let hosts = snapshot.getResources('Host');
    this._hostsCount = hosts.length;

    let kinds = ['AuthService', 'RateLimitService', 'TracingService', 'LogService'];
    let services = [];
    kinds.forEach((k)=>{
      services.push(...snapshot.getResources(k))
    });
    this._servicesCount = services.length;

    let mappings = snapshot.getResources('Mapping');
    this._mappingsCount = mappings.length;
	},

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClickHosts: function() {
    window.location.hash = "#hosts";
  },

  onClickMappings: function() {
    window.location.hash = "#mappings";
  },

  onClickServices: function() {
    window.location.hash = "#services";
  }
};

/**
 * Panel showing System Status
 */
let StatusPanel = {
  _title: "System Status",
  _elementId: "system_status",

  render: function() {
    let redis = this._snapshot.getRedisInUse();
    let envoy = this._diagd.envoy_status.ready;
    let errors= this._diagd.errors.length;

    const cos = Math.cos;
    const sin = Math.sin;
    const π = Math.PI;

    const f_matrix_times = (( [[a,b], [c,d]], [x,y]) => [ a * x + b * y, c * x + d * y]);
    const f_rotate_matrix = ((x) => [[cos(x),-sin(x)], [sin(x), cos(x)]]);
    const f_vec_add = (([a1, a2], [b1, b2]) => [a1 + b1, a2 + b2]);

    const f_svg_ellipse_arc = (([cx,cy],[rx,ry], [t1, Δ], φ ) => {
      /* [
      returns a SVG path element that represent a ellipse.
      cx,cy → center of ellipse
      rx,ry → major minor radius
      t1 → start angle, in radian.
      Δ → angle to sweep, in radian. positive.
      φ → rotation on the whole, in radian
      url: SVG Circle Arc http://xahlee.info/js/svg_circle_arc.html
      Version 2019-06-19
       ] */
      Δ = Δ % (2*π);
      const rotMatrix = f_rotate_matrix (φ);
      const [sX, sY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1), ry * sin(t1)] ), [cx,cy] ) );
      const [eX, eY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1+Δ), ry * sin(t1+Δ)] ), [cx,cy] ) );
      const fA = ( (  Δ > π ) ? 1 : 0 );
      const fS = ( (  Δ > 0 ) ? 1 : 0 );
      return "M " + sX + " " + sY + " A " + [ rx , ry , φ / (2*π) *360, fA, fS, eX, eY ].join(" ")
    });

    return html`
    <div class="element" style="cursor:pointer" @click=${this.onClick}>
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>
        <svg class="element-svg-overlay">
          <g stroke="${redis ? "#22EE55" : "red"}" fill="none" stroke-linecap="round" stroke-width="8">
            <path d="${f_svg_ellipse_arc([100,100], [90,90], [0,1.95], 0)}" />
            </g>
          <g stroke="${errors === 0 ? "#22EE55" : "red"}" fill="none" stroke-linecap="round" stroke-width="8">
            <path d="${f_svg_ellipse_arc([100,100], [90,90], [2.1,1.95], 0)}" />
            </g>
          <g stroke="${envoy ? "#22EE55" : "red"}" fill="none" stroke-linecap="round" stroke-width="8">
            <path d="${f_svg_ellipse_arc([100,100], [90,90], [4.2,1.95], 0)}" />
            </g>
        </svg>
        <div class="system-status">
        ${this.renderStatus(redis, "Redis in use", "Redis unavailable")}
        ${this.renderStatus(envoy, "Envoy ready", "Envoy unavailable")}
        ${this.renderStatus(errors === 0, "No Errors", countString(errors, "Error", "Errors"))}
        </div>
      </div>
    </div>`
  },

  renderStatus: function(condition, true_text, false_text) {
    return html`
      ${condition
      ? html`<p><span class = "status" style="color: green">${true_text}</span></p>`
      : html`<p><span class = "status" style="color: red">${false_text}</span></p>`}
    `;
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      this._snapshot  = snapshot;
      let diagnostics = snapshot.getDiagnostics();
      this._diagd = (('system' in (diagnostics||{})) ? diagnostics :
     {
       system: {
         env_status: {},
       },
       envoy_status: {},
       loginfo: {},
       errors: [],
     });
    }
  },

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClick: function() {
    window.location.hash = "#debugging";
  }
};

  /**
 * Panel showing System Services count and status
 */
let SystemServicesPanel = {
  _title: "System Services",
  _elementId: "system_services",

  render: function() {
    const cos = Math.cos;
    const sin = Math.sin;
    const π = Math.PI;

    const f_matrix_times = (( [[a,b], [c,d]], [x,y]) => [ a * x + b * y, c * x + d * y]);
    const f_rotate_matrix = ((x) => [[cos(x),-sin(x)], [sin(x), cos(x)]]);
    const f_vec_add = (([a1, a2], [b1, b2]) => [a1 + b1, a2 + b2]);

    const f_svg_ellipse_arc = (([cx,cy],[rx,ry], [t1, Δ], φ ) => {
      /* [
      returns a SVG path element that represent a ellipse.
      cx,cy → center of ellipse
      rx,ry → major minor radius
      t1 → start angle, in radian.
      Δ → angle to sweep, in radian. positive.
      φ → rotation on the whole, in radian
      url: SVG Circle Arc http://xahlee.info/js/svg_circle_arc.html
      Version 2019-06-19
       ] */
      Δ = Δ % (2*π);
      const rotMatrix = f_rotate_matrix (φ);
      const [sX, sY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1), ry * sin(t1)] ), [cx,cy] ) );
      const [eX, eY] = ( f_vec_add ( f_matrix_times ( rotMatrix, [rx * cos(t1+Δ), ry * sin(t1+Δ)] ), [cx,cy] ) );
      const fA = ( (  Δ > π ) ? 1 : 0 );
      const fS = ( (  Δ > 0 ) ? 1 : 0 );
      return "M " + sX + " " + sY + " A " + [ rx , ry , φ / (2*π) *360, fA, fS, eX, eY ].join(" ")
    });

    let redis = this._snapshot.getRedisInUse();
    let envoy = this._diagd.envoy_status.ready;
    let errors= this._diagd.errors.length;
    let stats = this._diagd.cluster_stats;

    /* Calculate number of running and waiting services,
     * and for running services, average health percentage
     */

    let services_running = 0;
    let services_waiting = 0;
    let services_pct_sum = 0;

    for (const [key, value] of Object.entries(stats)) {
      if (value.healthy_percent) {
        services_running += 1
        services_pct_sum += value.healthy_percent
      }
      else {
        services_waiting += 1
      }
    };

    /* Draw a circle of the average percentage. */
    let total_services   =  services_running + services_waiting;
    let average_health   = (services_running > 0 ? services_pct_sum/services_running : 100);
    const twopi  = 6.28; // real pi causes the ellipse to draw incorrectly at 2*pi
    const arcgap = 0.15;

    const health_radians = twopi*(average_health/100);

    /* Unfortunate hack: can't factor out the SVG code, so have to have a conditional
     * and duplicate code :-(  See renderArc for how we'd really like to switch between the two conditions:
     * ${renderArc("green", 0, health_radians)}
       ${average_health < 100 ? renderArc("red", health_radians+arcgap, twopi-health_radians-2*arcgap) : html``}
     */

    var result;

   result = html`
      <div class="element" style="cursor:pointer" @click=${this.onClick}>
        <div class="element-titlebar">${this._title}</div>
        <div class="element-content" id=“${this._elementId}”>
          <svg class="element-svg-overlay">
            ${renderArc("#22EE55", 0, health_radians)}
            ${average_health < 100 ? renderArc("red", health_radians+arcgap, twopi-health_radians-2*arcgap) : html``}
         </svg>
          <div class="system-status">
          <p><span class = "status" style="color: green">${countString(total_services, "Service", "Services")}</span></p>
          <p><span class = "status" style="color: ${average_health >= 80  ? "green" : "gray"}">${average_health}% Healthy</span></p>
          <p><span class = "status" style="color: ${services_waiting == 0 ? "green" : "gray"}">${services_waiting} Waiting</span></p>
  
          </div>
        </div>
      </div>`;



    return result;
  },

  renderStatus: function(condition, true_text, false_text) {
    return html`
      ${condition
      ? html`<p><span class = "status" style="color: green">${true_text}</span></p>`
      : html`<p><span class = "status" style="color: red">${false_text}</span></p>`}
    `;
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      this._snapshot  = snapshot;
      let diagnostics = snapshot.getDiagnostics();
      this._diagd = (('system' in (diagnostics||{})) ? diagnostics :
     {
       system: {
         env_status: {},
       },
       envoy_status: {},
       loginfo: {},
       errors: [],
     });
    }
  },

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClick: function() {
    window.location.hash = "#debugging";
  }
};


/* ===================================================================================*/
/* The Dashboard class, drawing dashboard elements in a matrix of div.element blocks. */
/* ===================================================================================*/

export class Dashboard extends LitElement {
  /* styles() returns the styles for the dashboard elements. */
  static get styles() {
    return css`
      .error {
        color: red;
      }

      div.element {
        display: inline-grid;
        padding-bottom: 0.5em;
        padding-right: 0.5em;
      }

      div.element-titlebar {
        text-align: center;
        font-weight: bold;
        background-color: lightgray;
        width: 200px;
        height: 16px;
        border: 2px solid lightgray;
        padding: 8px;
        left-margin: 20px;
        bottom-margin: 0px;
      }

      div.element-content {
        position: relative;
        background-color: whitesmoke;
        text-align: center;
        width: 200px;
        height: 200px;
        border: 2px solid lightgray;
        padding: 8px;
        top-margin: 0px;
        left-margin: 20px;
      }
      div.centered {
        display: flex;
        justify-content: center;
        align-items: center;
      }

      div.element-content p {
        margin-top: 0.5em;
        margin-bottom: 0.5em;
      }
      
      span.code { font-family: Monaco, monospace; }
      span.status {
        font-family: Helvetica;
        font-weight: 900;
        font-size: 130%;
        color: #555555;
      }
      
      svg.element-svg-overlay {
        position:absolute;
        height:200px;
        width:200px;
        top:0.5em;
        left:0.5em;
      }
      
      div.system-status {
        font-size: 90%;
        padding-top: 65px;
      }
      
      div.system-status p {
        margin: 0;
      }
`
  };

  static get properties() {
    return {
      snapshot: { type: Object }
    };
  }

  constructor() {
    super();

    /* Initialize the list of dashboard panels */
    this._panels = [ CountsPanel, StatusPanel, SystemServicesPanel, LicensePanel ];

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
  };

  /* Get new data from Kubernetes services. */
  onSnapshotChange(newSnapshot) {
    /* Notify each panel of the change */
    this._panels.forEach((panel) => {
      panel.onSnapshotChange(newSnapshot)
    });

    this.snapshot = newSnapshot;
  }

  /* Render the component by returning a TemplateResult, using the html helper function. */
  render() {
    /*
     * Wait for the update to be completed and then..
     */
    this.updateComplete.then(() => {
      WhenChartsAreLoadedPromise.then(
        () => {
          /* ..and then draw the charts for each panel */
          this._panels.forEach((panel) => {
            panel.draw(this.shadowRoot);
          });
        });
    });

    /*
     * Return the concatenated html renderings for each panel
     */
    return( this._panels.reduce( (accum, each) => html`${accum} ${each.render()}`, html`` ) );
  }

  /*
   * chartsLoaded callback.  Resolve the promise to let the Dashboard
   * know that the library is available and charts can now be drawn.
   * It is a sort of mutex collaborating with updateComplete which is called
   * in the render() method.
   */
  chartsLoaded() {
    WhenChartsAreLoadedPromise.resolve();
  }
}

/* ===================================================================================*/

customElements.define('dw-dashboard', Dashboard);
