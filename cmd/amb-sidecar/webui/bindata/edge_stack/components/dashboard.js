
  /* Dashboard and dashboard element classes. */

import { LitElement, html, css } from "https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js";
/* import { Chart } from "https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.5.0/Chart.min.js"; */

export class Dashboard extends LitElement {

  /* styles() returns the styles for frames, triangles, etc. copied from resources.js.
     this should really be in a superclass that is shared by all Admin pages. */
  static get styles() {
    return css`
      .error {
        color: red;
      }
      div {
        margin: 0.4em;
      }
      div.frame {
        display: grid;
        grid-template-columns: 50% 50%;
        border: 2px solid #ede7f3;
        border-radius: 0.4em;
      }
      div.title {
        grid-column: 1 / 3;
        background: #ede7f3;
        margin: 0;
        padding: 0.5em;
      }

      div.dash-element {
        background-color: lightgrey;
        width: 300px;
        border: 15px solid green;
        padding: 50px;
        margin: 20px;
      }

      /*
      * These styles are used in mappings.js
      */
      div.frame-no-grid {
        border: 2px solid #ede7f3;
        border-radius: 0.4em;
      }
      .collapsed div.up-down-triangle {
        float: left;
        margin-left: 0;
        margin-top: 0.25em;
        cursor: pointer;
      }
      .collapsed div.up-down-triangle::before {
        content: "\\25B7"
      }
      .expanded div.up-down-triangle {
        float: left;
        margin-left: 0;
        margin-top: 0.25em;
        cursor: pointer; 
      }
      .expanded div.up-down-triangle::before {
        content: "\\25BD"
      }
      div.grid {
        display: grid;
        grid-template-columns: 50% 50%;
      }
      div.grid div {
        margin: 0.1em;
      }
      /*
      * End of styles for mappings.js
      */
      
      div.left {
        grid-column: 1 / 2;
      }
      div.right {
        grid-column: 2 / 3;
      }
      div.both {
        grid-column: 1 / 3;
      }
      .off { display: none; }
      span.code { font-family: Monaco, monospace; }`
  };

  /* The constructor doesn't do anything at the moment...*/
  constructor() {
    super()
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

  /* Render the component by returning a TemplateResult, using the html helper function. */
  render() {
    return html`<p>Hello World from the Dashboard</p>`
  };

  /* Returns a single graph item in a box. */
  renderGraphItem() {
    return html`<div class="dash-element">Graph Item</div>`
  }

  /* Returns a single summary item in a box. */
  renderSummaryItem() {
    return html`<div class="dash-element">Summary Item</div>`
  }
};

customElements.define('dw-dashboard', Dashboard);
