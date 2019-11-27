
/* ===================================================================================*/
/* Dashboard and dashboard element classes using LitElement and Google Charts. */
/* ===================================================================================*/

import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import { Snapshot } from '/edge_stack/components/snapshot.js'

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

/*
 * Load the google charts package.  Note that it will load asynchronously so we have
 * to wait for a setOnLoadCallback which is done in the Dashboard constructor.
 */
google.charts.load('current', {'packages':['corechart']});
google.charts.load('visualization', '1.0', { 'packages': ['corechart'] });

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
      return "M " + sX + " " + sY + " L " + (sX + 100) + " " + sY;
      //return "M " + sX + " " + sY + " A " + [ rx , ry , φ / (2*π) *360, fA, fS, eX, eY ].join(" ")
    });

    var result = html`
        <g stroke="${color}" fill="none" stroke-linecap="round" stroke-width="8">
            <path d="${f_svg_ellipse_arc([100,100], [90,90], [start_rad,end_rad], 0)}"/>
        </g>
    `;
    result = html`<g stroke="red" fill="blue" stroke-width="18">
<path d="M 50 50 L 50 100 L 100 100 Z"></path>
</g>`

    return result
  };

/**
 * Pie Chart Demo - this is a demo only and not an Ambassador-useful panel.
 */
let demoPieChart = {
  _title: "Demo Pie Chart",
  _elementId: "demo_pie",

  render: function() {
    return html`<div class="element">
        <div class="element-titlebar">${this._title}</div>
        <div class="element-content" id="${this._elementId}"></div>
     </div>`
  },

  onSnapshotChange: function(snapshot) {
    this._mushrooms  = Math.random(20);
    this._pepperoni  = Math.random(10);
	},

  draw: function(shadow_root) {
    /* Create the data table. */
    let data = new google.visualization.DataTable();
    data.addColumn('string', 'Topping');
    data.addColumn('number', 'Slices');
    data.addRows([
      ['Mushrooms', this._mushrooms],
      ['Onions', 1],
      ['Olives', 1],
      ['Zucchini', 1],
      ['Pepperoni', this._pepperoni]
    ]);

    /* Set chart options */
    let options = {
      'title': 'How Much Pizza I Ate Last Night',
      'width': '80%',
      'height': '80%'
    };

    /* Instantiate and draw our chart, passing in some options. */
    let element = shadow_root.getElementById(this._elementId);

    /* may not have shadow DOM by now, so test. */
    if (element) {
      if( element.offsetParent !== null ) {
        let chart = new google.visualization.PieChart(element);
        chart.draw(data, options);
      }
    }
  }
};


/**
 * Column Chart Demo - this demo has no useful Ambassador purpose
 */
let demoColumnChart = {
  _title: "Demo Column Chart",
  _elementId: "demo_columns",
  _annualRevenue: [
        ['Year', 'Millions', { role: 'style' } ],
        [2015, 10, 'color: gray'],
        [2016, 11, 'color: gray'],
        [2017, 13, 'color: gray'],
        [2018, 15, 'color: gray'],
        [2019, 16, 'color: gray']
  ],

  render: function() {
    return html`<div class="element">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id="${this._elementId}"></div>
    </div>`
  },

  onSnapshotChange: function(services) {
    /* Revenue data: keep 5 columns, and bump revenue by
     * a random amount between -1 and 3. */
    for (let i=1; i<=4; i++) {
      this._annualRevenue[i] = this._annualRevenue[i+1]
    }
    /* Bump up the year and revenue number */
    this._annualRevenue[5] = [
      this._annualRevenue[4][0] + 1,
      this._annualRevenue[4][1] + Math.floor(Math.random() * Math.floor(3)) - 1,
      this._annualRevenue[4][2]
    ];
	},

  draw: function(shadow_root) {
    /* Create the data table. */
    let data = google.visualization.arrayToDataTable(this._annualRevenue);

    /* Set chart options */
    let options = {
      'title': 'Annual Revenue',
      'width': '80%',
      'height': '80%'
    };

    /* Instantiate and draw our chart, passing in some options. */
    let element = shadow_root.getElementById(this._elementId);

    /* may not have shadow DOM by now, so test. */
    if (element) {
      if( element.offsetParent !== null ) {
        let chart = new google.visualization.ColumnChart(element);
        chart.draw(data, options);
      }
    }
  }
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
      <div class="element-content" id=“${this._elementId}”>
        <p style="margin-top: 2.8em"><span class = "status" @click=${this.onClickHosts}>${countString(this._hostsCount, "Host", "Hosts")}</span></p>
        <p><span class = "status" @click=${this.onClickMappings}>${countString(this._mappingsCount, "Mapping", "Mappings")}</span></p>
        <p><span class = "status" @click=${this.onClickServices}>${countString(this._servicesCount, "Service", "Services")}</span></p>
      </div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      let hosts = snapshot.getResources('Host');
      this._hostsCount = hosts.length;

      let kinds = ['AuthService', 'RateLimitService', 'TracingService', 'LogService'];
      let services = [];
      kinds.forEach((k)=>{
        services.push(...snapshot.getResources(k))
      });
      this._serviceCount = services.length;

      let mappings = snapshot.getResources('Mapping');
      this._mappingsCount = mappings.length;
    }
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
 * Panel showing Cluster count and status
 */
let ClustersPanel = {
  _title: "Clusters",
  _elementId: "clusters_status",

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

    /* Calculate number of running and waiting clusters,
     * and for running clusters, average health percentage
     */

    let clusters_running = 0;
    let clusters_waiting = 0;
    let clusters_pct_sum = 0;

    for (const [key, value] of Object.entries(stats)) {
      if (value.healthy_percent) {
        clusters_running += 1
        clusters_pct_sum += value.healthy_percent
      }
      else {
        clusters_waiting += 1
      }
    };

    /* Draw a circle of the average percentage. */
    let total_clusters   =  clusters_running + clusters_waiting;
    let average_health   = (clusters_running > 0 ? clusters_pct_sum/clusters_running : 100);
    const twopi  = 6.28; // real pi causes the ellipse to draw incorrectly at 2*pi
    const arcgap = 0.15;

    const health_radians = twopi*(average_health/100);

    /* Unfortunate hack: can't factor out the SVG code, so have to have a conditional
     * and duplicate code :-(  See renderArc for how we'd really like to switch between the two conditions:
     * ${renderArc("green", 0, health_radians)}
       ${average_health < 100 ? renderArc("red", health_radians+arcgap, twopi-health_radians-2*arcgap) : html``}
     */

    var result;

    if (average_health == 100) {
      result = html`
      <div class="element" style="cursor:pointer" @click=${this.onClick}>
        <div class="element-titlebar">${this._title}</div>
        <div class="element-content" id=“${this._elementId}”>
          <svg class="element-svg-overlay">
             <g stroke="green" fill="none" stroke-linecap="round" stroke-width="8">
                <path d="${f_svg_ellipse_arc([100,100], [90,90], [0, twopi], 0)}"/>
             </g>
          </svg>
          <div class="system-status">
          <p><span class = "status" style="color: green">${countString(total_clusters, "Cluster", "Clusters")}</span></p>
          <p><span class = "status" style="color: ${average_health >= 80  ? "green" : "gray"}">${average_health}% Healthy</span></p>
          <p><span class = "status" style="color: ${clusters_waiting == 0 ? "green" : "gray"}">${clusters_waiting} Waiting</span></p>
  
          </div>
        </div>
      </div>`;
    }
    else {
      result = html`
      <div class="element" style="cursor:pointer" @click=${this.onClick}>
        <div class="element-titlebar">${this._title}</div>
        <div class="element-content" id=“${this._elementId}”>
          <svg class="element-svg-overlay">
             <g stroke="green" fill="none" stroke-linecap="round" stroke-width="8">
                <path d="${f_svg_ellipse_arc([100,100], [90,90], [0, health_radians], 0)}"/>
             </g>
             <g stroke="red" fill="none" stroke-linecap="round" stroke-width="8">
                <path d="${f_svg_ellipse_arc([100,100], [90,90], [health_radians+arcgap, twopi-health_radians-2*arcgap], 0)}"/>
            </g>
          </svg>
          <div class="system-status">
          <p><span class = "status" style="color: green">${countString(total_clusters, "Cluster", "Clusters")}</span></p>
          <p><span class = "status" style="color: ${average_health >= 80  ? "green" : "gray"}">${average_health}% Healthy</span></p>
          <p><span class = "status" style="color: ${clusters_waiting == 0 ? "green" : "gray"}">${clusters_waiting} Waiting</span></p>
  
          </div>
        </div>
      </div>`;
    };

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

  constructor() {
    super();

    /* Initialize the list of dashboard panels */
    this._panels = [StatusPanel, CountsPanel, ClustersPanel];

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
    /* Set up the Google Charts setOnLoad callback.  Note that we can't draw
     * charts until the package has loaded, so we will be notified when this
     * happens and set a promise to synchronize with the DOM being updated. */
    google.charts.setOnLoadCallback(this.chartsLoaded.bind(this));
  };

  /* Get new data from Kubernetes services. */
  onSnapshotChange(snapshot) {
    /* Notify each panel of the change */
    this._panels.forEach((panel) => {
      panel.onSnapshotChange(snapshot)
    });

    /* Request an update of the Dashboard */
    this.requestUpdate();
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
