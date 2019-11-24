
  /* Dashboard and dashboard element classes.
   * Assumes that ChartJS has been loaded into the html page.
   */

import { LitElement, html, css } from "https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js";
import {useContext, registerContextChangeHandler} from '/edge_stack/components/context.js'

   // Set a callback to run when the Google Visualization API is loaded.
  google.charts.setOnLoadCallback(chartsLoaded);

/* The Dashboard class, drawing dashboard elements in a matrix of dash-element div's. */
export class Dashboard extends LitElement {
  /* styles() returns the styles for frames, triangles, etc. copied from resources.js.
     this should really be in a superclass that is shared by all Admin pages. */
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

  /* The constructor doesn't do anything at the moment...*/
  constructor() {
    super()

    const [currentSnapshot, setSnapshot] = useContext('aes-api-snapshot', null);
    this.onSnapshotChange(currentSnapshot);
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));

  };

  /* Initialize the dashboard. */
  init() {
    super.init()
  };

  /* Reset the dashboard. */
  reset() {
    super.reset()
  };

  /* Update, will eventually call render() */
  update() {
    super.update()
  };

  /* Validate the dashboard.  Not sure why this would be called.*/
  validate() {
    this.state.messages.push("validating dashboard...why?")
  };

  onSnapshotChange(snapshot) {
    this.services = [
    (((snapshot || {}).Kubernetes || {}).AuthService || []),
    (((snapshot || {}).Kubernetes || {}).RateLimitService || []),
    (((snapshot || {}).Kubernetes || {}).TracingService || []),
    (((snapshot || {}).Kubernetes || {}).LogService || []),
  ].reduce((acc, item) => acc.concat(item));

  this.requestUpdate();
}

/* Render the component by returning a TemplateResult, using the html helper function. */
  render() {
    /* return html`Hello World from Dashboard` */
    var test_value = "testing...";
    var num_of_services = 0;

    if (this.services) {
      num_of_services = this.services.length
    }

    return html`
      ${this.renderSummaryItem("Number of Services", num_of_services)}
      ${this.renderSummaryItem("Summary 1", test_value)}
      ${this.renderGraphItem("Graph 1", test_value)}
      ${this.renderSummaryItem("Summary 1", test_value)}
      ${this.renderGraphItem("Graph 1", test_value)}
      ${this.renderSummaryItem("Summary 1", test_value)}
    `
  };

  updated(changedProperties) {
    changedProperties.forEach((oldValue, propName) => {
      console.log(`${propName} changed. oldValue: ${oldValue}`);
    });
  }

  firstUpdated() {
    this.drawChart("chart-div");
  }

  /* Draw a chart in the given element. */
  drawChart(element_id) {
    if (true) {
      // Create the data table.
      var data = new google.visualization.DataTable();
      data.addColumn('string', 'Topping');
      data.addColumn('number', 'Slices');
      data.addRows([
        ['Mushrooms', 3],
        ['Onions', 1],
        ['Olives', 1],
        ['Zucchini', 1],
        ['Pepperoni', 2]
      ]);

      // Set chart options
      var options = {'title':'How Much Pizza I Ate Last Night',
                     'width':400,
                     'height':300};

      // Instantiate and draw our chart, passing in some options.
      var element  = this.shadowRoot.getElementById(element_id);
      var chart    = new google.visualization.PieChart(element);
      chart.draw(data, options);
    }

  }

  /* Returns a single graph item in a box. */
  renderGraphItem(title, value) {
    return html`
        <div class="element">
        <div class="element-titlebar">Graph ${title}</div>
        <div class="element-content">Graph Content Goes Here: ${value}</div>
        </div>`
  }

  /* Returns a single summary item in a box. */
  renderSummaryItem(title, value) {
    return html`
        <div class="element">
        <div class="element-titlebar">${title}</div>
        <div class="element-content">${value}</div>
        </div>`
  }
};

// Callback to recognize when the Google Charts are loaded.
function chartsLoaded() {
  console.log("Google Charts Loaded")
}

customElements.define('dw-dashboard', Dashboard);
