import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Signup extends LitElement {

  static get styles() {
    return css`
#signup {
    margin: auto;
    width: 50%;
}

#signup-content, #signup-finished {
    display: none;
    background-color: #fefefe;
    padding: 20px;
    border: 1px solid #888;
    width: 80%;
}

#signup-error {
    color: #fe0000;
}

.invalid {
    background-color: #fe0000;
}
`
  }

  static get properties() {
    return {
      state: { type: String },
      message: { type: String }
    }
  }

  constructor() {
    super()
    this.state = "start"
    this.message = ""
  }

  handleSignup() {
    this.state = "entry"
    this.email().focus()
  }

  reset() {
    this.state = "start"
    this.message = ""
    this.email().value = ""
    this.confirm().value = ""
  }

  handleSubmit() {
    if (this.email().value == "") {
      this.email().classList.add("invalid")
      this.message = "Please supply an email."
    } else if (this.email().value != this.confirm().value) {
      this.email().classList.remove("invalid")
      this.confirm().classList.add("invalid")
      this.message = "Emails do not match."
    } else {
      console.log("blah")
      this.email().classList.remove("invalid")
      this.confirm().classList.remove("invalid")
      this.message = "Requesting a license key..."
      this.state = "pending"

      fetch("https://metriton.datawire.io/signup", {
        method: "POST",
        headers:{
          "content-type": "application/json; charset=UTF-8"
        },
        body: JSON.stringify({
          email: this.email().value,
          confirm: this.confirm().value
        })
      })
        .then(data=>{return data.json()})
        .then(res=>{
          console.log(res);
          if ("vid" in res) {
            this.message = "Congratulations! A license key has been sent to " + this.email().value
          } else {
            this.message = "Sorry, there was a problem processing your request. Please contact support@datawire.io and supply this:<br>" + btoa(JSON.stringify(res))
          }
        })
        .catch(error=>console.log(error))
    }
  }

  email() {
    return this.shadowRoot.getElementById("email")
  }

  confirm() {
    return this.shadowRoot.getElementById("confirm")
  }

  render() {
    return html`
<button @click=${this.handleSignup} style="display:${this.state == "start" ? "block" : "none"}">
  Click here to sign up for a free Community license.
</button>

<div style="display:${this.state == "entry" ? "block" : "none"}">
  <form>
    <span>
      Email:
    </span>
    <input id="email" type="text" name="email" value="" /> 
    <span>
      Confirm:
    </span>
    <input id="confirm" type="text" name="email-confirm" value="" /> <input @click=${this.reset} type="button" value="Cancel" /> <input @click=${this.handleSubmit} type="button" value="Signup" /> 
  </form>
  <div class="invalid">${this.message}</div>
</div>

<div style="display:${this.state == "pending" ? "block" : "none"}">
  <span>${this.message}</span>
  <button @click=${this.reset}>Continue</button>
</div>
`
  }

}

customElements.define('dw-signup', Signup)
