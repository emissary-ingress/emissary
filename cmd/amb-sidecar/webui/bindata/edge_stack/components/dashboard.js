
/* ===================================================================================*/
/* Dashboard and dashboard element classes using LitElement and Google Charts. */
/* ===================================================================================*/

import { LitElement, html, css } from "https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js";  //TODO FIXME
import {useContext, registerContextChangeHandler} from '/edge_stack/components/context.js'

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
 *        Return a string of the html for the panel. Note that it must return a string
 *        and not an html template.
 *   onSnapshotChange(snapshot) => (no return value)
 *        Called each time a new snapshot is received from Ambassador. This function
 *        is used to update any internal state for the panel from the data in the snapshot.
 *   draw(shadow_root) => (no return value)
 *        Use the google chart apis to create a chart and attach it to panel's element
 *        in the shadow DOM. This function does nothing if there is no chart in the panel.
 */

/**
 * Pie Chart Demo - this is a demo only and not an Ambassador-useful panel.
 */
let demoPieChart = {
  _title: "Demo Pie Chart",
  _elementId: "demo_pie",

  render: function() {
    return `<div class="element">
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
    var data = new google.visualization.DataTable();
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
    var options = {
      'title': 'How Much Pizza I Ate Last Night',
      'width': '80%',
      'height': '80%'
    };

    /* Instantiate and draw our chart, passing in some options. */
    var element = shadow_root.getElementById(this._elementId);

    /* may not have shadow DOM by now, so test. */
    if (element) {
      var chart = new google.visualization.PieChart(element);
      chart.draw(data, options);
    }
  }
};

/**
 * Panel showing a count of the Services
 */
let demoServiceCount = {
  _title: "Number of Services",
  _elementId: "demo_services",
  _serviceCount: 0,

  render: function() {
    return `<div class="element">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>${this._serviceCount}</div>
    </div>`
  },

  onSnapshotChange: function(snapshot) {
    if (snapshot) {
      this._services = [
      (((snapshot || {}).Kubernetes || {}).AuthService || []),
      (((snapshot || {}).Kubernetes || {}).RateLimitService || []),
      (((snapshot || {}).Kubernetes || {}).TracingService || []),
      (((snapshot || {}).Kubernetes || {}).LogService || []),
    ].reduce((acc, item) => acc.concat(item));

      this._serviceCount = this._services.length;
    }
	},

  draw: function(shadow_root) {}
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
    return `<div class="element">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id="${this._elementId}"></div>
    </div>`
  },

  onSnapshotChange: function(services) {
    /* Revenue data: keep 5 columns, and bump revenue by
     * a random amount between -1 and 3. */
    var i;
    for (i=1; i<=4; i++) {
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
    var data = google.visualization.arrayToDataTable(this._annualRevenue);

    /* Set chart options */
    var options = {
      'title': 'Annual Revenue',
      'width': '80%',
      'height': '80%'
    };

    /* Instantiate and draw our chart, passing in some options. */
    var element = shadow_root.getElementById(this._elementId);

    /* may not have shadow DOM by now, so test. */
    if (element) {
      var chart = new google.visualization.ColumnChart(element);
      chart.draw(data, options);
    }
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
       
      span.code { font-family: Monaco, monospace; }`
  };

  constructor() {
    super();

    /* Initialize the list of dashboard panels */
    this._panels = [ demoPieChart,  demoServiceCount, demoColumnChart ]

    const [currentSnapshot, setSnapshot] = useContext('aes-api-snapshot', null);
    this.onSnapshotChange(currentSnapshot);
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));

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
    let the_html = this._panels.reduce((html_str, panel) => {
      return html_str + panel.render() + "\n";
    }, "");
    return html([the_html]);
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
