
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
      case 0:  return `No ${plural_text}`;    break;
      case 1:  return `One ${singular_text}`; break;
      default: return `${count} ${plural_text}`;
    }
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
 * Panel showing a count of the Hosts
 */
let HostsPanel = {
  _title: "Hosts",
  _elementId: "hosts_count",
  _hostsCount: 0,

  render: function() {
    return html`
    <div class="element" style="cursor:pointer" @click=${this.onClick}>
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>
        <br><br><br>
        <p><span class = "status">${countString(this._hostsCount, "Host", "Hosts")}</span></p>
      </div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      let hosts = snapshot.getResources('Host');
      this._hostsCount = hosts.length;
    }
	},

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClick: function() {
    window.location.hash = "#hosts";
  }
};


/**
 * Panel showing a count of the Services
 */
let ServicesPanel = {
  _title: "Services",
  _elementId: "services_count",
  _serviceCount: 0,

  render: function() {
    return html`
    <div class="element" style="cursor:pointer" @click=${this.onClick}>
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>
        <br><br><br>
        <p><span class = "status">${this._serviceCount} Services</span></p>
      </div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      let kinds = ['AuthService', 'RateLimitService', 'TracingService', 'LogService'];
      let services = [];
      kinds.forEach((k)=>{
        services.push(...snapshot.getResources(k))
      });
      this._serviceCount = services.length;
    }
	},

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClick: function() {
    window.location.hash = "#services";
  }
};

/**
 * Panel showing a count of the Mappings
 */
let MappingsPanel = {
  _title: "Mappings",
  _elementId: "mappings_count",
  _mappingsCount: 0,

  render: function() {
    return html`
    <div class="element" style="cursor:pointer" @click=${this.onClick}>
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>
        <br><br><br>
        <p><span class = "status">${this._mappingsCount} Mappings</span></p>
      </div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      let mappings = snapshot.getResources('Mapping');
      this._mappingsCount = mappings.length;
    }
	},

  draw: function(shadow_root) { /*text panel, no chart to draw*/ },

  onClick: function() {
    window.location.hash = "#mappings";
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

    return html`
    <div class="element" style="cursor:pointer" @click=${this.onClick}>
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>
        <br>
        ${this.renderStatus(redis, "Redis In Use", "Redis Unavailable")}
        ${this.renderStatus(envoy, "Envoy Ready", "Envoy Unavailable")}
        ${this.renderStatus(errors == 0, "No Errors", countString(errors, "Error", "Errors"))}
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
        background-color: whitesmoke;
        text-align: center; 
        width: 200px;
        height: 200px;
        border: 2px solid lightgray;
        padding: 8px;
        top-margin: 0px;
        left-margin: 20px;
       }
       
      span.code { font-family: Monaco, monospace; }
      span.status { font-family: Helvetica; font-weight: 900; font-size: 150%;}`
  };

  constructor() {
    super();

    /* Initialize the list of dashboard panels */
    this._panels = [ HostsPanel, MappingsPanel, ServicesPanel, demoPieChart, StatusPanel, demoColumnChart ];

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
