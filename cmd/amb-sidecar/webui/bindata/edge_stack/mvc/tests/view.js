import {Model} from "../framework/model2.js";
import {View, html} from "../framework/view2.js";

describe('View', function() {
  it('should render when non-model properties are set', function(done) {
    let v = new MyView()
    let c = new Counter()
    document.body.appendChild(v)
    v.onRender = ()=>{
      if (v.name === "My View") {
        v.remove()
        done()
      } else {
        c.once(()=>{v.name = "My View"})
      }
    }
  })

  it('should render when model properties are set', function(done) {
    let v = new MyView()
    let m = new MyModel()
    let c = new Counter()
    document.body.appendChild(v)
    v.onRender = ()=>{
      if (v.model === m) {
        v.remove()
        done()
      } else {
        c.once(()=>{v.model = m})
      }
    }
  })

  it('should render when models are updated', function(done) {
    let v = new MyView()
    let m = new MyModel()
    v.model = m
    let c = new Counter()
    document.body.appendChild(v)
    let notified = false
    v.onRender = ()=>{
      if (notified) {
        v.remove()
        done()
      } else {
        c.once(()=>{
          notified = true
          m.notify()
        })
      }
    }
  })

})

class MyModel extends Model {}

class MyView extends View {
  static get properties() {
    return {
      name: {type: String},
      model: {type: MyModel}
    }
  }

  constructor() {
    super()
    this.name = ""
    this.model = null
  }

  render() {
    this.onRender()
    return html`(${this.name}, ${this.model})`
  }
}

customElements.define('my-view', MyView)

// helper for the view tests
class Counter {
  constructor() {
    this.calls = 0
  }

  once(f) {
    if (this.calls === 0) {
      setTimeout(f, 1)
    }
    this.calls++
  }
}
