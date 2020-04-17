import { View, html, css } from '../framework/view2.js'
import 'https://cdnjs.cloudflare.com/ajax/libs/prism/1.19.0/prism.min.js'
import 'https://cdnjs.cloudflare.com/ajax/libs/prism/1.19.0/components/prism-javascript.min.js'

class TextInclude extends View {

  static get properties() {
    return {
      src: { type: String },
      section: { type: String },
    }
  }

  constructor() {
    super()
    this.src = ""
    this.section = ""
  }

  parse(text) {
    if (!this.section) {
      return text
    }

    let section = []
    let appending = false

    for (let line of text.split("\n")) {
      let trimmed = line.trim()

      if (trimmed.startsWith("//SECTION:")) {
        if (appending) {
          return section.join("\n")
        } else {
          let parts = trimmed.split(":", 2)
          if (parts[1] === this.section) {
            appending = true
            continue
          }
        }
      }

      if (appending) {
        section.push(line)
      }
    }

    if (appending) {
      return section.join("\n")
    }

    return `unknown section: ${this.section}`
  }

  connectedCallback() {
    super.connectedCallback()
    fetch(this.src)
      .then((r)=>r.text())
      .then((t)=>{
        let section = this.parse(t)
        section = section.replace(/</g, "&lt;")
        section = section.replace(/>/g, "&gt;")
        this.innerHTML = `
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.19.0/themes/prism.min.css" integrity="sha256-cuvic28gVvjQIo3Q4hnRpQSNB0aMw3C+kjkR0i+hrWg=" crossorigin="anonymous">
<pre><code class="lang-js">${section}</code></pre>
`
        Prism.highlightAllUnder(this)
      })
  }

  render() {
    return html`<slot></slot>`
  }

}

customElements.define('text-include', TextInclude)
