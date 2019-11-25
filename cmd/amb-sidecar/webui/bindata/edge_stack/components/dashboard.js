
/* ===================================================================================*/
/* Dashboard and dashboard element classes using LitElement and Google Charts. */
/* ===================================================================================*/

/* Import the LitElement class and html and css helpers. */
import { LitElement, html, css } from "https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js";  //TODO FIXME

/* Get context updates */
import {useContext, registerContextChangeHandler} from '/edge_stack/components/context.js'

/* Create a global Promise to sychronize charts loaded and shadow dom finalized. */

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

/* Load the charts package.  Note that it will load asynchronously so we have
 * to wait for a setOnLoadCallback (see the Dashboard constructor)
 */
google.charts.load('current', {'packages':['corechart']});
google.charts.load('visualization', '1.0', { 'packages': ['corechart'] });

/* ===================================================================================
 * Dashboard panels defined as objects.  Declared here so that they can be added
 * to the Dashboard on instantiation.
 * ===================================================================================
 */

/* Pie Chart Demo */
let demoPieChart = {
  _title: "Demo Pie Chart",
  _elementId: "demo_pie",

  render: function() {
    return `
    <div class="element">
        <div class="element-titlebar">${this._title}</div>
        <div class="element-content" id="${this._elementId}"></div>
     </div>
    `
  },

  onSnapshotChange: function(snapshot) {
    this._mushrooms  = Math.random(20);
    this._pepperoni  = Math.random(10);
	},

  draw: function(dashboard) {
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
    var element = dashboard.shadowRoot.getElementById(this._elementId);

    /* may not have shadow DOM by now, so test. */
    if (element) {
      var chart = new google.visualization.PieChart(element);
      chart.draw(data, options);
    }
  }
};

/* Count of Services Demo */
let demoServiceCount = {
  _title: "Number of Services",
  _elementId: "demo_services",
  _serviceCount: 0,

  render: function() {
    return `
    <div class="element">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id=“${this._elementId}”>${this._serviceCount}</div>
    </div>
    `
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

  draw: function(dashboard) {}
};

/* Column Chart Demo */
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
    return `
    <div class="element">
      <div class="element-titlebar">${this._title}</div>
      <div class="element-content" id="${this._elementId}"></div>
    </div>
    `
  },

  onSnapshotChange: function(services) {
    /* Revenue data: keep 5 columns, and bump revenue by
     * a random amount between -1 and 3.
     */
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

  draw: function(dashboard) {
    /* Create the data table. */
    var data = google.visualization.arrayToDataTable(this._annualRevenue);

    /* Set chart options */
    var options = {
      'title': 'Annual Revenue',
      'width': '80%',
      'height': '80%'
    };

    /* Instantiate and draw our chart, passing in some options. */
    var element = dashboard.shadowRoot.getElementById(this._elementId);

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
        display: inline-block;
      }
      
      div.element-titlebar {
        text-align: center;
        font-weight: bold;
        background-color: lightgray;
        width: 240px;
        height: 16px;
        border: 2px solid lightgray;
        padding: 8px;
        left-margin: 20px;
        bottom-margin: 0px;
      }

      div.element-content {
        background-color: whitesmoke;
        text-align: center; 
        width: 240px;
        height: 240px;
        border: 2px solid lightgray;
        padding: 8px;
        top-margin: 0px;
        left-margin: 20px;
       }
       
      span.code { font-family: Monaco, monospace; }`
  };

  /* Dashboard constructor.  Set up callbacks, context handlers. */
  constructor() {
    super();

    /* Initialize the list of dashboard panels */
    this._panels = [ demoPieChart,  demoServiceCount, demoColumnChart ]

    /* Use the aes-api-snapshot and set up the callback. */
    const [currentSnapshot, setSnapshot] = useContext('aes-api-snapshot', null);
    this.onSnapshotChange(currentSnapshot);
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));

    /* Set up the Google Charts setOnLoad callback.  Note that we can't draw
     * charts until the package has loaded, so we will be notified when this
     * happens and set a promise to synchronize with the DOM being updated.
     */
    google.charts.setOnLoadCallback(this.chartsLoaded.bind(this));

    /* Watch for the DOMContentLoaded event, for debugging purposes. */
    document.addEventListener('DOMContentLoaded', (event) => { this.domLoaded() });
  };

  /* Updated properties from the Dashboard.  Currently this is not used. */
  updated(changedProperties) {
    changedProperties.forEach((oldValue, propName) => {
      console.log(`${propName} changed. oldValue: ${oldValue}`);
    });
  }

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
    /* Wait for the update to be completed */
    this.updateComplete.then(() => {
      WhenChartsAreLoadedPromise.then(
        () => {
          console.log("draw charts");
          /* Notify each panel to draw */
          this._panels.forEach((panel) => {
            panel.draw(this)
          });
        });
    });

    /* Return the concatenated html renderings for each panel */
    var the_html = this._panels.reduce((html_str, panel) => {
      return html_str + panel.render() + "\n";
    }, "");

    return html([the_html]);
  }


  /* chartsLoaded callback.  Resolve the promise to let the Dashboard
   * know that the library is available and charts can now be drawn.
   * It is a sort of mutex collaborating with updateComplete which is called
   * in the render() method.
   */
  chartsLoaded() {
    WhenChartsAreLoadedPromise.resolve();
  }
}

/* ===================================================================================*/

/* Define the Dashboard lit-element class as 'dw-dashboard' */
customElements.define('dw-dashboard', Dashboard);
