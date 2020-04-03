//SECTION:Clock
import { Model } from '../edge_stack/mvc/framework/model.js'

class Clock extends Model {

  constructor() {
    super()
    this.now = new Date()
    this.paused = false

    // Update the current time every second.
    setInterval(()=>{
      if (!this.paused) {
        this.now = new Date()
      }
    }, 1000)
  }

  // Our getter and setter for the 'now' field.
  get now() {
    return this._now
  }

  // We call this.notify() in the setter in order to ensure any Views
  // are updated.
  set now(value) {
    this._now = value
    // notify any views that our state has changed
    this.notify()
  }

  pause() {
    this.paused = true
    this.notify()
  }

  unpause() {
    this.paused = false
    this.notify()
  }

  // padded accessors

  get hours() {
    return pad(this.now.getHours())
  }

  get minutes() {
    return pad(this.now.getMinutes())
  }

  get seconds() {
    return pad(this.now.getSeconds())
  }

}
//SECTION:ignored

function pad(n) { return n.toString().padStart(2, 0) }

//SECTION:global
let CLOCK = new Clock()

//SECTION:Digital
import { View, html, css } from '../edge_stack/mvc/framework/view.js'

class Digital extends View {

  // Define the clock as a normal lit-element property.
  static get properties() {
    return {
      clock: { type: Model }
    }
  }

  constructor() {
    super()
    // Initialize the property to our shared Clock model.
    this.clock = CLOCK
  }

  // Render the clock's current state.
  render() {
    return html`<h1>${this.clock.hours}:${this.clock.minutes}:${this.clock.seconds}</h1>`
  }

}

customElements.define('dw-digital', Digital)

//SECTION:Analog
class Analog extends View {

  static get properties() {
    return {
      clock: { type: Model }
    }
  }

  static get styles() {
    return clockCSS()
  }

  constructor() {
    super()
    this.clock = CLOCK
  }

  render() {
    return html`
<svg viewBox="0 0 40 40"
     style="--hours: ${this.clock.hours}; --minutes: ${this.clock.minutes}; --seconds: ${this.clock.seconds}">
  <circle cx="20" cy="20" r="19" />
  <g class="marks">
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
    <line x1="15" y1="0" x2="16" y2="0" />
  </g>
  <line x1="0" y1="0" x2="9" y2="0" class="hour" />
  <line x1="0" y1="0" x2="13" y2="0" class="minute" />
  <line x1="0" y1="0" x2="16" y2="0" class="seconds" />
  <circle cx="20" cy="20" r="0.7" class="pin" />
</svg>
`
  }

}

customElements.define('dw-analog', Analog)
//SECTION:ignored

//SECTION:Control
class Control extends View {

  // Define the clock as a normal lit-element property.
  static get properties() {
    return {
      clock: { type: Model }
    }
  }

  constructor() {
    super()
    this.clock = CLOCK
  }

  // This is our click handler, it modifies the model by
  // pausing/un-pausing. Try loading the clock.html page and pressing
  // the button. You will see both the digital and analog clock
  // rendered and responding to the toggle button.
  toggle() {
    console.log("toggle")
    if (this.clock.paused) {
      this.clock.unpause()
    } else {
      this.clock.pause()
    }
  }

  // This render also depends on being notified of updates from the
  // model, because it changes the label of the button based on the
  // state of the model. Again, because we extended the Model and View
  // classes, this notification happens automatically.
  render() {
    return html`<button @click=${()=>this.toggle()}>${this.clock.paused ? "Start" : "Stop"}</button>`
  }

}

customElements.define('dw-control', Control)
//SECTION:ignored

function clockCSS() {
    return css`
svg {
  height: 250px;
  fill: none;
  stroke: #000;
  stroke-width: 1;
  stroke-linecap: round;
  transform: rotate(-90deg);
}

circle {
  fill: white;
}

.marks {
  transform: translate(20px, 20px);
  stroke-width: 0.2;
}

.seconds,
.minute,
.hour
{
  transform: translate(20px, 20px) rotate(0deg);
}

.seconds {
  transform: translate(20px, 20px) rotate(calc(var(--seconds) * 6deg));
  stroke-width: 0.3;
  stroke: #d00505;
}

.minute {
  transform: translate(20px, 20px) rotate(calc(var(--minutes) * 6deg));
  stroke-width: 0.6;
}

.hour {
  transform: translate(20px, 20px) rotate(calc(var(--hours) * 30deg));
  stroke-width: 1;
}

.pin {
  stroke: #d00505;
  stroke-width: 0.2;
}

/* marks */
.marks > line:nth-child(1) {
  transform: rotate(30deg);
}

.marks > line:nth-child(2) {
  transform: rotate(calc(2 * 30deg));
}

.marks > line:nth-child(3) {
  transform: rotate(calc(3 * 30deg));
  stroke-width: 0.5;
}

.marks > line:nth-child(4) {
  transform: rotate(calc(4 * 30deg));
}
.marks > line:nth-child(5) {
  transform: rotate(calc(5 * 30deg));
}

.marks > line:nth-child(6) {
  transform: rotate(calc(6 * 30deg));
  stroke-width: 0.5;
}

.marks > line:nth-child(7) {
  transform: rotate(calc(7 * 30deg));
}

.marks > line:nth-child(8) {
  transform: rotate(calc(8 * 30deg));
}

.marks > line:nth-child(9) {
  transform: rotate(calc(9 * 30deg));
  stroke-width: 0.5;
}

.marks > line:nth-child(10) {
  transform: rotate(calc(10 * 30deg));
}

.marks > line:nth-child(11) {
  transform: rotate(calc(11 * 30deg));
}
.marks > line:nth-child(12) {
  transform: rotate(calc(12 * 30deg));
  stroke-width: 0.5;
}
`
}
