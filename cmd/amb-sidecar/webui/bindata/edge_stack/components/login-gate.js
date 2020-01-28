import {LitElement, css, html} from '../vendor/lit-element.min.js'
import {registerContextChangeHandler, useContext} from './context.js'
import {ApiFetch, hasDebugBackend} from './api-fetch.js'
import {updateCredentials} from './snapshot.js'

export class LoginPage extends LitElement {
  static get styles() {
    return css`
article,
aside,
details,
figcaption,
figure,
footer,
header,
hgroup,
main,
nav,
section,
summary {
    display: block
}

audio,
canvas,
video {
    display: inline-block
}

audio:not([controls]) {
    display: none;
    height: 0
}

[hidden] {
    display: none
}

html {
    -ms-text-size-adjust: 100%;
    -webkit-text-size-adjust: 100%
}

body,
figure {
    margin: 0
}

a:focus {
    outline: thin dotted
}

a:active,
a:hover {
    outline: 0
}

h1 {
    font-size: 2em;
    margin: .67em 0
}

abbr[title] {
    border-bottom: 1px dotted
}

b,
strong {
    font-weight: 700
}

dfn {
    font-style: italic
}

hr {
    box-sizing: content-box;
    height: 0
}

mark {
    background: #ff0;
    color: #000
}

code,
kbd,
pre,
samp {
    font-family: monospace, serif;
    font-size: 1em
}

pre {
    white-space: pre-wrap
}

q {
    quotes: "\\201C" "\\201D" "\\2018" "\\2019"
}

small {
    font-size: 80%
}

sub,
sup {
    font-size: 75%;
    line-height: 0;
    position: relative;
    vertical-align: baseline
}

sup {
    top: -.5em
}

sub {
    bottom: -.25em
}

img {
    border: 0
}

svg:not(:root) {
    overflow: hidden
}

fieldset {
    border: 1px solid silver;
    margin: 0 2px;
    padding: .35em .625em .75em
}

legend {
    border: 0;
    padding: 0
}

button,
input,
select,
textarea {
    font-family: inherit;
    font-size: 100%;
    margin: 0
}

button,
input {
    line-height: normal
}

button,
select {
    text-transform: none
}

button,
html input[type=button],
input[type=reset],
input[type=submit] {
    -webkit-appearance: button;
    cursor: pointer
}

button[disabled],
html input[disabled] {
    cursor: default
}

input[type=checkbox],
input[type=radio] {
    box-sizing: border-box;
    padding: 0
}

input[type=search] {
    -webkit-appearance: textfield;
    box-sizing: content-box
}

input[type=search]::-webkit-search-cancel-button,
input[type=search]::-webkit-search-decoration {
    -webkit-appearance: none
}

button::-moz-focus-inner,
input::-moz-focus-inner {
    border: 0;
    padding: 0
}

*,
textarea {
    vertical-align: top
}

textarea {
    overflow: auto
}

table {
    border-collapse: collapse;
    border-spacing: 0
}

* {
    margin: 0;
    padding: 0;
    border: 0;
    position: relative;
    box-sizing: border-box
}

.alpha .col_left .logo,
body,
html,
navigation a,
navigation a .label,
navigation a .label .icon {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center
}

body,
html {
    height: 100%;
    color: #000;
    font-family: 'Source Sans Pro', sans-serif;
    font-size: 16px;
    background: #000;
    -webkit-font-smoothing: antialiased
}

a {
    color: #333;
    text-decoration: none
}

.login .content_con .card_con .card .cta_download:hover,
a:hover {
    color: #5f3eff
}

.alpha,
.alpha .col_left,
.alpha .col_right {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex
}

.alpha {
    padding: 0;
    margin: auto;
    max-width: 1440px;
    position: relative;
    -webkit-flex: 1 1 auto;
    -ms-flex: 1 1 auto;
    flex: 1 1 auto;
    min-height: 100%
}

.alpha .col_left,
.alpha .col_right {
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column
}

.alpha .col_left {
    background: #2e3147;
    -webkit-flex: 0 0 250px;
    -ms-flex: 0 0 250px;
    flex: 0 0 250px
}

.alpha .col_left .logo,
navigation a,
navigation a .label,
navigation a .label .icon {
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center
}

.alpha .col_left .logo {
    -webkit-flex: 0 0 80px;
    -ms-flex: 0 0 80px;
    flex: 0 0 80px;
    background: #5f3eff;
    padding: 0
}

.alpha .col_left .logo img {
    width: 90%;
    max-width: 175px
}

.alpha .col_right {
    -webkit-flex: 3 0 auto;
    -ms-flex: 3 0 auto;
    flex: 3 0 auto;
    background: #f3f3f3
}

navigation {
    display: block;
    width: 100%
}

navigation a {
    padding: 0;
    text-decoration: none;
    height: 60px;
    transition: all .9s ease
}

navigation a .selected_stripe {
    -webkit-flex: 0 0 10px;
    -ms-flex: 0 0 10px;
    flex: 0 0 10px;
    background: #ff4329;
    min-height: 100%;
    opacity: 0
}

navigation a,
navigation a .label,
navigation a .label .icon {
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center
}

navigation a .label {
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row;
    margin-left: 6%;
    -webkit-flex: 3 0 0;
    -ms-flex: 3 0 0px;
    flex: 3 0 0
}

navigation a .label .icon {
    height: 100%;
    -webkit-flex: 0 0 25px;
    -ms-flex: 0 0 25px;
    flex: 0 0 25px
}

navigation a .label .icon svg,
navigation a.selected .label .icon svg {
    width: 100%;
    height: auto;
    max-height: 35px
}

navigation a .label .icon svg path,
navigation a .label .icon svg polygon,
navigation a .label .icon svg rect {
    fill: #9a9a9a;
    transition: fill .7s ease
}

navigation a .label .name {
    -webkit-flex: 1 0 auto;
    -ms-flex: 1 0 auto;
    flex: 1 0 auto;
    color: #9a9a9a;
    padding-left: 20px;
    font-size: 1rem;
    transition: all .7s ease
}

navigation a:hover {
    background: #363a58;
    transition: all .8s ease
}

navigation a:hover .label .icon svg path,
navigation a:hover .label .icon svg polygon,
navigation a:hover .label .icon svg rect {
    fill: #53f7d2;
    transition: fill .7s ease
}

navigation a:hover .label .name {
    color: #53f7d2;
    transition: all .7s ease
}

navigation a.selected {
    background: #5f3eff;
    transition: all 2.8s ease
}

navigation a.selected .selected_stripe {
    -webkit-flex: 0 0 10px;
    -ms-flex: 0 0 10px;
    flex: 0 0 10px;
    background: #ff4329;
    min-height: 100%;
    opacity: 1
}

.content,
navigation a.selected .label {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex
}

navigation a.selected .label,
navigation a.selected .label .icon {
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center;
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center
}

navigation a.selected .label {
    margin-left: 6%;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row;
    -webkit-flex: 3 0 0;
    -ms-flex: 3 0 0px;
    flex: 3 0 0
}

navigation a.selected .label .icon {
    height: 100%;
    -webkit-flex: 0 0 25px;
    -ms-flex: 0 0 25px;
    flex: 0 0 25px;
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center
}

navigation a.selected .label .icon svg path,
navigation a.selected .label .icon svg polygon,
navigation a.selected .label .icon svg rect {
    fill: #fff;
    transition: fill .7s ease
}

navigation a.selected .label .name {
    -webkit-flex: 1 0 auto;
    -ms-flex: 1 0 auto;
    flex: 1 0 auto;
    color: #fff;
    padding-left: 20px;
    font-size: 1rem;
    transition: all .7s ease
}

.content {
    width: 100%;
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column;
    max-width: 900px;
    margin: 0 auto;
    padding: 30px
}

.content .header_con,
.content a.button_large,
navigation a.selected .label {
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center
}

.content a.button_large {
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center;
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center;
    background: #55acd1;
    border-radius: 15px;
    padding: 20px;
    width: 450px;
    color: #fff;
    transition: all .3s ease
}

.content a.button_large:hover {
    color: #fff;
    background: #2296c7;
    transition: all .9s ease
}

.content .header_con,
.content .header_con .col,
.content a.button_large {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex
}

.content .header_con {
    margin: 30px 0 0;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row
}

.content .header_con .col {
    -webkit-flex: 0 0 80px;
    -ms-flex: 0 0 80px;
    flex: 0 0 80px;
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column
}

.content .header_con .col svg {
    width: 100%;
    height: 60px
}

.content .header_con .col svg path,
.content .header_con .col svg stroke,
.login .content_con .card_con .card .cta_download:hover svg .cls-1 {
    fill: #5f3eff
}

.content .header_con .col:nth-child(2) {
    -webkit-flex: 2 0 0;
    -ms-flex: 2 0 0px;
    flex: 2 0 0;
    padding-left: 20px
}

.content .header_con .col h1 {
    padding: 0;
    margin: 0;
    font-weight: 400
}

.content .header_con .col p {
    margin: 0;
    padding: 0
}

.content .card,
.content .card .col .con {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex
}

.content .card {
    background: #fff;
    border-radius: 10px;
    padding: 30px;
    box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
    width: 100%;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row;
    -webkit-flex: 1 1 1;
    -ms-flex: 1 1 1;
    flex: 1 1 1;
    margin: 30px 0 0
}

.content .card .col {
    -webkit-flex: 1 0 50%;
    -ms-flex: 1 0 50%;
    flex: 1 0 50%;
    padding: 0 30px 0 0
}

.content .card .col .con {
    margin: 10px 0;
    -webkit-flex: 1;
    -ms-flex: 1;
    flex: 1;
    -webkit-justify-content: flex-end;
    -ms-flex-pack: end;
    justify-content: flex-end;
    height: 30px
}

.content .card .col .con .pararen,
.content .card .col .con label {
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center;
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center
}

.content .card .col .con label {
    display: inline-block;
    text-transform: uppercase;
    margin: 0 10px 0 0;
    padding: 0;
    font-size: 1rem
}

.content .card .col .con .bold {
    font-weight: 600
}

.content .card .col .con input {
    background: #efefef;
    padding: 5px
}

.content .card .col .con input.mapping {
    width: 80%
}

.content .card .col .con input.prefix,
.content .card .col .con input.services {
    width: 40%
}

.content .card .col .con .pararen {
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center;
    font-size: 1.2rem;
    padding: 0 5px
}

.content .card .col,
.content .card .col .con label,
.content .card .col2,
.content .card .col2 a.cta .label {
    -webkit-align-self: center;
    -ms-flex-item-align: center;
    -ms-grid-row-align: center;
    align-self: center
}

.content .card .col2 a.cta {
    text-decoration: none;
    border: 2px #efefef solid;
    border-radius: 10px;
    width: 90px;
    padding: 6px 8px;
    -webkit-flex: auto;
    -ms-flex: auto;
    flex: auto;
    margin: 10px auto;
    color: #000;
    transition: all .2s ease
}

.content .card .col2 a.cta .label {
    text-transform: uppercase;
    font-size: .8rem;
    font-weight: 600;
    line-height: 1rem;
    padding: 0 0 0 10px;
    -webkit-flex: 1 0 auto;
    -ms-flex: 1 0 auto;
    flex: 1 0 auto
}

.content .card .col2 a.cta svg {
    width: 15px;
    height: auto
}

.content .card .col2 a.cta svg path,
.content .card .col2 a.cta svg polygon,
.content .card .col2 a.cta svg stroke {
    transition: fill .7s ease;
    fill: #000
}

.content .card .col2 a.cta:hover {
    color: #5f3eff;
    transition: all .2s ease;
    border: 2px #5f3eff solid
}

.content .card .col2 a.cta:hover svg path,
.content .card .col2 a.cta:hover svg polygon,
.content .card .col2 a.cta:hover svg stroke {
    transition: fill .2s ease;
    fill: #5f3eff
}

.content .card .col2 a.cta,
.content .card_small,
.content .row {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row
}

.content .card_small {
    -webkit-flex: 1;
    -ms-flex: 1;
    flex: 1;
    background: #fff;
    border-radius: 10px;
    margin: 30px 0 0;
    box-shadow: 0 10px 15px -20px rgba(0, 0, 0, .8);
    padding: 20px;
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column;
    font-size: .9rem
}

.content .card_small .col {
    -webkit-flex: 1;
    -ms-flex: 1;
    flex: 1;
    padding: 10px 5px
}

.content .card_small .col:nth-child(1) {
    font-weight: 600
}

.content .card_small .col:nth-child(2) {
    -webkit-flex: 1.5;
    -ms-flex: 1.5;
    flex: 1.5
}

.content .card_small .row {
    -webkit-flex: 1 100%;
    -ms-flex: 1 100%;
    flex: 1 100%;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row
}

.content .card_small .row:nth-last-child(1) {
    border-bottom: none
}

.content .card_small .line {
    border-bottom: 1px solid rgba(0, 0, 0, .1)
}

.content .card_small .copy_button,
.content .card_small .image {
    width: 100%;
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center
}

.content .card_small .image {
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center;
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center
}

.content .card_small .copy_button a,
.content .card_small .image a {
    display: block;
    width: 100%;
    text-align: center;
    font-size: 2rem
}

.content .card_small .copy_button a img,
.content .card_small .image a img {
    width: 50%;
    margin: auto
}

.content .card_small .copy_button {
    padding: 0 5px;
    height: 100px
}

.content .card_small .copy_button a {
    font-size: 4rem
}

.content .card_small .row_tall {
    -webkit-flex: 3;
    -ms-flex: 3;
    flex: 3
}

.content .margin-right {
    margin-right: 20px
}

.footer,
.login {
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column
}

.footer {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    height: 100px
}

.login {
    height: 100%;
    max-width: 1440px
}

.login .header {
    width: 100%;
    height: 52px;
    text-align: right;
    background: #2e3147;
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex
}

.login .header .logo,
.login .hero,
.login .hero .description .cta {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    -webkit-justify-content: center;
    -ms-flex-pack: center;
    justify-content: center;
    -webkit-align-content: center;
    -ms-flex-line-pack: center;
    align-content: center
}

.login .header .logo {
    background: #5f3eff;
    padding: 0;
    -webkit-flex: 0 0 250px;
    -ms-flex: 0 0 250px;
    flex: 0 0 250px
}

.login .header .logo img {
    width: 85%;
    max-width: 165px
}

.login .hero,
.login .hero .description .cta {
    -webkit-align-items: center;
    -ms-flex-align: center;
    align-items: center
}

.login .hero {
    -webkit-flex: 1 auto;
    -ms-flex: 1 auto;
    flex: 1 auto;
    background: #fff url(../images/blackbird.svg) no-repeat
}

.login .hero .description {
    text-align: center
}

.login .hero .description h1 {
    color: #5f3eff;
    font-size: 2.5rem
}

.login .hero .description p {
    font-weight: 500
}

.login .content_con p span,
.login .hero .description p span {
    font-weight: 800
}

.login .hero .description .cta {
    padding: 20px;
    width: 80%;
    margin: auto
}

.login .hero .description .cta .copy {
    -webkit-flex: 2 0 0;
    -ms-flex: 2 0 0px;
    flex: 2 0 0;
    color: #6b46ff;
    border-radius: 10px;
    background: #e8e8e8;
    padding: 10px;
    margin-right: 10px
}

.login .content_con .card_con .card a.button,
.login .hero .description .cta a.button {
    -webkit-flex: 1 0 0;
    -ms-flex: 1 0 0px;
    flex: 1 0 0;
    border-radius: 5px;
    border: 2px solid #efefef;
    padding: 10px;
    text-transform: uppercase;
    font-size: .8rem;
    font-weight: 700;
    color: #6e6e6e;
    transition: all .3s ease
}

.login .content_con .card_con .card a.button svg,
.login .hero .description .cta a.button svg {
    margin-right: 3px
}

.login .hero .description .cta a.button svg path {
    fill: #333;
    transition: fill .3s ease
}

.login .content_con .card_con .card a.button:hover,
.login .hero .description .cta a.button:hover {
    border: 2px solid #5f3eff;
    color: #5f3eff;
    transition: all .4s ease
}

.login .content_con .card_con .card a.button:hover svg path,
.login .hero .description .cta a.button:hover svg path {
    fill: #5f3eff;
    transition: fill .4s ease
}

.login .content_con,
.login .content_con .card_con {
    display: -webkit-flex;
    display: -ms-flexbox;
    display: flex;
    padding: 30px
}

.login .content_con {
    background: #f3f3f3;
    height: 345px;
    text-align: center;
    -webkit-flex-direction: column;
    -ms-flex-direction: column;
    flex-direction: column;
    -webkit-flex: auto;
    -ms-flex: auto;
    flex: auto
}

.login .content_con .card_con {
    -webkit-flex: 1 0 0;
    -ms-flex: 1 0 0px;
    flex: 1 0 0;
    -webkit-flex-direction: row;
    -ms-flex-direction: row;
    flex-direction: row
}

.login .content_con .card_con .card {
    padding: 20px;
    text-align: left;
    -webkit-flex: 1 1 200px;
    -ms-flex: 1 1 200px;
    flex: 1 1 200px
}

.login .content_con .card_con .card .subtitle {
    font-weight: 600;
    padding: 10px 0
}

.login .content_con .card_con .card .subtitle2 {
    font-weight: 600;
    margin-top: 30px;
    margin-bottom: 10px
}

.login .content_con .card_con .card .subtitle3 {
    margin-top: 10px;
    margin-bottom: 10px
}

.login .content_con .card_con .card .subtitle4 {
    margin-top: -12px;
    margin-bottom: 10px
}

.login .content_con .card_con .card .card_option1 {
    margin: 1px;
    padding: 3px 0px 10px 10px;
    border: 1px #ccc;
    border-style: none none none solid;
}

.login .content_con .card_con .card .card_option2 {
    margin: 1px;
    padding: 3px 0px 0px 10px;
    border: 1px #ccc;
    border-style: none none none solid;
}

.login .content_con .card_con .card .card_copy {
    -webkit-hyphens: auto;
    -ms-hyphens: auto;
    hyphens: auto;
    font-size: .8rem;
    background: #e9e9e9;
    border-radius: 10px;
    padding: 12px;
    width: 90%;
    margin-bottom: 20px
}

.login .content_con .card_con .card .title {
    font-weight: 800;
    font-size: 1.5rem
}

.login .content_con .card_con .card .cta_download {
    color: #37a4ff;
    font-weight: 600
}

.login .content_con .card_con .card .cta_download svg {
    height: 20px
}

.login .content_con .card_con .card .cta_download svg .cls-1 {
    fill: #37a4ff
}

.login .content_con .card_con .card a.button {
    width: 180px;
    border: 2px solid #ccc;
    padding: 8px;
    text-align: center
}

.login .content_con .card_con .card a.button svg path {
    fill: #6e6e6e;
    transition: fill .3s ease
}
`;
  }

  copyToKeyboard(theId) {
    const copyText = this.shadowRoot.getElementById(theId).innerText;
    const el = document.createElement('textarea');  // Create a <textarea> element
    el.value = copyText;                            // Set its value to the string that you want copied
    el.setAttribute('readonly', '');                // Make it readonly to be tamper-proof
    el.style.position = 'absolute';
    el.style.left = '-9999px';                      // Move outside the screen to make it invisible
    document.body.appendChild(el);                  // Append the <textarea> element to the HTML document
    const selected =
      document.getSelection().rangeCount > 0        // Check if there is any content selected previously
        ? document.getSelection().getRangeAt(0)     // Store selection if found
        : false;                                    // Mark as false to know no selection existed before
    el.select();                                    // Select the <textarea> content
    document.execCommand('copy');                   // Copy - only works as a result of a user action (e.g. click events)
    document.body.removeChild(el);                  // Remove the <textarea> element
    if (selected) {                                 // If a selection existed before copying
      document.getSelection().removeAllRanges();    // Unselect everything on the HTML document
      document.getSelection().addRange(selected);   // Restore the original selection
    }
  }

  copyLoginToKeyboard() {
    this.copyToKeyboard('login-cmd');
  }

  copyDarwinInstallToKeyboard() {
    this.copyToKeyboard('install-darwin');
  }

  copyLinuxInstallToKeyboard() {
    this.copyToKeyboard('install-linux');
  }

  copyWindowsInstallToKeyboard() {
    this.copyToKeyboard('install-windows');
  }

  static get properties() {
    return {
      namespace: String
    };
  }

  constructor() {
    super();
    this.namespace = this.getAttribute("namespace");
  }

  render() {
    return html`
    <div class="alpha login">
    
        <div class="header">
            <div class="logo"><img src="../images/ambassador-logo-white.svg"></div>
        </div>
        
        <div class="hero">
            <div class="description">
                <h1>Welcome to the Ambassador Edge Stack</h1>
                <p><span>Repeat users</span> can log in to the Edge Policy Console directly with this command:</p>
                <div class="cta">
                    <div class="copy" id="login-cmd">edgectl login --namespace=${this.namespace} ${window.location.host}</div>
                    <a class="button" href="#" @click=${this.copyLoginToKeyboard.bind(this)}><?xml version="1.0" encoding="UTF-8"?>
<svg width="16px" height="15px" viewBox="0 0 16 15" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-640.000000, -241.000000)" fill="#505050" fill-rule="nonzero">
            <g id="cta-buttons" transform="translate(628.000000, 229.000000)">
                <path d="M27,16 L27,26 L16,26 L16,16 L27,16 Z M28,15 L15,15 L15,27 L28,27 L28,15 Z M12,25 L12,12 L26,12 L26,13.2380952 L13.3333333,13.2380952 L13.3333333,25 L12,25 Z" id="Shape"></path>
            </g>
        </g>
    </g>
</svg> Copy to Clipboard</a>
                </div>
            </div>
        </div>
        
        <div class="content_con">
            <p><span>First time users</span> will need to download and install the edgectl executable.
Once complete, log in to Ambassador with the edgectl command</p>
            
            <div class="card_con">
                <div class="card">
                    <div class="title">MacOS <?xml version="1.0" encoding="UTF-8"?>
<svg width="17px" height="21px" viewBox="0 0 17 21" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-143.000000, -441.000000)" fill="#000000" fill-rule="nonzero">
            <g id="Group-4" transform="translate(69.000000, 436.000000)">
                <g id="Apple_logo_black" transform="translate(74.000000, 5.000000)">
                    <path d="M16.3695747,15.7731054 C16.0693844,16.4802346 15.7140567,17.1311429 15.3023668,17.7295795 C14.7411939,18.5454005 14.2817188,19.1101049 13.9276164,19.4236909 C13.3786968,19.9384208 12.7905683,20.2020323 12.1607806,20.2170248 C11.708657,20.2170248 11.1634128,20.085843 10.5287243,19.8197339 C9.89195299,19.5548709 9.30676503,19.4236909 8.77169058,19.4236909 C8.21051806,19.4236909 7.60866646,19.5548709 6.96491104,19.8197339 C6.32017466,20.085843 5.80078406,20.2245211 5.40367465,20.2382639 C4.79974024,20.2644999 4.19776675,19.9933921 3.59689511,19.4236909 C3.21338674,19.0826192 2.7336946,18.4979254 2.15904376,17.6696095 C1.54248936,16.7850747 1.03559616,15.7593626 0.638486917,14.5899759 C0.213196572,13.3268881 0,12.1037796 0,10.919651 C0,9.56323739 0.287447796,8.39335082 0.863200991,7.41298979 C1.31569229,6.62552819 1.91766622,6.00435375 2.6710829,5.54834262 C3.42450026,5.09233218 4.23856805,4.85995409 5.11524722,4.84508656 C5.59493902,4.84508656 6.22399188,4.99638224 7.00571234,5.2937264 C7.7852276,5.59206995 8.28574951,5.74336563 8.5051946,5.74336563 C8.66925788,5.74336563 9.22528432,5.56645808 10.1678827,5.21376852 C11.0592653,4.88668997 11.8115793,4.75126053 12.4278888,4.8046075 C14.0979288,4.94203554 15.3526027,5.61330869 16.18701,6.82267442 C14.693409,7.74544067 13.9545725,9.03788775 13.9692756,10.6958931 C13.9827536,11.9873407 14.4422287,13.0620269 15.3452509,13.91533 C15.75449,14.3113717 16.2115152,14.6174612 16.72,14.8348475 C16.6097275,15.1609263 16.493327,15.4732626 16.3695747,15.7731054 Z M12.5393886,0.404915241 C12.5393886,1.41714633 12.1767089,2.36226362 11.4538015,3.23705439 C10.5814103,4.2770095 9.52621045,4.87794407 8.38193279,4.78311899 C8.36735228,4.66168248 8.35889782,4.53387476 8.35889782,4.39957035 C8.35889782,3.42782958 8.77377356,2.38787498 9.51052727,1.53757083 C9.87835194,1.10704695 10.3461592,0.749072171 10.9134582,0.463508902 C11.4795317,0.182206408 12.0149738,0.0266400704 12.5185584,0 C12.5332626,0.135319484 12.5393886,0.270647484 12.5393886,0.404902113 L12.5393886,0.404915241 Z" id="path4"></path>
                </g>
            </g>
        </g>
    </g>
</svg></div>
                    <div class="subtitle">Download with this CLI</div>
                    <div class="card_option1">
                       <div class="card_copy" id="install-darwin">sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl</div>
                       <a href="#" class="button" @click=${this.copyDarwinInstallToKeyboard.bind(this)}><?xml version="1.0" encoding="UTF-8"?>
<svg width="16px" height="15px" viewBox="0 0 16 15" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-640.000000, -241.000000)" fill="#505050" fill-rule="nonzero">
            <g id="cta-buttons" transform="translate(628.000000, 229.000000)">
                <path d="M27,16 L27,26 L16,26 L16,16 L27,16 Z M28,15 L15,15 L15,27 L28,27 L28,15 Z M12,25 L12,12 L26,12 L26,13.2380952 L13.3333333,13.2380952 L13.3333333,25 L12,25 Z" id="Shape"></path>
            </g>
        </g>
    </g>
</svg> Copy to Clipboard</a>
                    </div>
                    <div class="subtitle2">Or download the binary:</div>
                    <div class="card_option2">
                        <a href="https://metriton.datawire.io/downloads/darwin/edgectl" class="cta_download"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 19 20"><defs><style>.cls-1{fill:#2d8fff;}</style></defs><title>download</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><g id="Screen"><g id="login"><g id="Group-4"><g id="iconmonstr-download-19"><path id="Shape" class="cls-1" d="M9.5,17.5,2.59,10H7.77V0h3.46V10h5.18Zm7.77-.83v1.66H1.73V16.67H0V20H19V16.67Z"/></g></g></g></g></g></g></svg> Download edgectl for MacOS</a>
                        <div class="subtitle3">make it executable:</div>
                        <div class="card_copy">chmod a+x ~/Downloads/edgectl</div>
                        <div class="subtitle4">and place it somewhere in your shell PATH.</div>
                    </div>
                </div>
                <div class="card">
                    <div class="title">Linux <?xml version="1.0" encoding="UTF-8"?>
<svg width="22px" height="26px" viewBox="0 0 22 26" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-418.000000, -436.000000)" fill-rule="nonzero">
            <g id="Group-4" transform="translate(69.000000, 436.000000)">
                <g id="Tux_Mono" transform="translate(349.000000, 0.000000)">
                    <path d="M20.3521968,20.1331726 C20.3518988,20.1328751 20.351571,20.1325478 20.351571,20.13228 C20.1707021,19.928491 20.0845443,19.5506633 19.9918896,19.1481428 C19.8994733,18.7459198 19.7960898,18.3121616 19.4652867,18.0310221 C19.4646608,18.0303973 19.4637668,18.0298321 19.4632005,18.0292073 C19.3976359,17.9720869 19.3302832,17.9239213 19.2626325,17.8837287 C19.1947135,17.8435659 19.1258409,17.8108407 19.0570278,17.784958 C19.5168741,16.423588 19.3365714,15.0679003 18.8722549,13.843054 C18.302737,12.3394779 17.3082413,11.0296055 16.548586,10.1335289 C15.6983324,9.06281853 14.8668243,8.04655105 14.8832453,6.54535499 C14.9085472,4.25429062 15.1356392,0.00566183588 11.09686,0 C10.9326505,-0.000288207936 10.7612885,0.00655434245 10.582774,0.0208344476 C6.06954564,0.383489618 7.26669556,5.14352467 7.19964086,6.73724391 C7.11708907,7.90285749 6.88046048,8.82154426 6.07729418,9.96097765 C5.13405806,11.0807759 3.80548093,12.8934567 3.17635883,14.7805131 C2.87953002,15.6709372 2.73826812,16.5785866 2.86850325,17.437773 C2.82767439,17.4743657 2.78893167,17.5127733 2.75167906,17.5520435 C2.47481765,17.8474632 2.2700773,18.2050311 2.04209132,18.4457104 C1.82900638,18.6581567 1.52562112,18.7387798 1.19183772,18.8580484 C0.85805432,18.977674 0.491786646,19.1537656 0.269463061,19.5797887 C0.269463061,19.5797887 0.269463061,19.580116 0.26916504,19.580116 C0.268867019,19.580711 0.268568998,19.5816035 0.267972957,19.5821687 C0.163367624,19.7770327 0.129393242,19.9873667 0.129393242,20.2006758 C0.129393242,20.3979198 0.15859929,20.597871 0.188103358,20.7903251 C0.249495661,21.1907631 0.311782028,21.5694536 0.229230241,21.8259303 C-0.0348162685,22.5467781 -0.06879065,23.0450942 0.117174386,23.4068867 C0.303735463,23.7692443 0.686692308,23.929003 1.11971666,24.0194437 C1.98576537,24.19973 3.15847758,24.1551047 4.08264036,24.6441983 L4.16221194,24.4945547 L4.08353442,24.6444958 C5.07296378,25.1609596 6.0761021,25.3442507 6.87628819,25.1618521 C7.45683288,25.0297909 7.92770589,24.6846883 8.16969886,24.1539147 C8.79554273,24.1509396 9.48248088,23.8861627 10.582774,23.8257995 C11.3293163,23.7656743 12.2618237,24.0905467 13.3344009,24.0310463 C13.3624149,24.1470721 13.4029755,24.2589329 13.4583776,24.3651412 C13.4590034,24.366004 13.4595697,24.367194 13.4601955,24.3681162 C13.8759049,25.1981474 14.648375,25.5777601 15.4718365,25.5129047 C16.2961325,25.4480194 17.1726119,24.9628231 17.8813354,24.1211894 L17.7513983,24.0123036 L17.8822593,24.1199994 C18.5575746,23.3024634 19.6784312,22.9636084 20.4219933,22.5161651 C20.7936254,22.2924435 21.0949245,22.0122261 21.1184681,21.6052134 C21.1416542,21.1985279 20.9023434,20.7427545 20.3521968,20.1331726 Z" id="path3" fill="#202020"></path>
                    <path d="M20.2390468,20.9914593 C20.2232023,21.2461875 20.0215152,21.4354754 19.6488398,21.640731 C18.9041161,22.051544 17.5840726,22.4089302 16.7413771,23.3425413 C16.009197,24.1391141 15.1166244,24.5764896 14.3307051,24.6332367 C13.5447857,24.6899839 12.8667408,24.3917596 12.4666677,23.6579709 L12.4659745,23.6564616 L12.4650172,23.6546204 C12.2167539,23.2229799 12.32,22.5423162 12.5290223,21.8239518 C12.7379714,21.1058591 13.0383564,20.3681464 13.0786278,19.7689812 L13.0786278,19.7671399 C13.1208467,18.9992727 13.1680831,18.3285397 13.309,17.8108731 C13.45028,17.2932064 13.6724328,16.9430645 14.0659372,16.7459588 C14.0844224,16.7369034 14.1025445,16.7281499 14.1206996,16.72 C14.1652953,17.3849677 14.5254272,18.0635487 15.1618142,18.2102159 C15.8583443,18.3780727 16.8621583,17.8317005 17.2859982,17.3858733 C17.3707992,17.3828246 17.4533226,17.3789308 17.5335683,17.3771197 C17.9055835,17.3689397 18.2171917,17.3885899 18.5360619,17.6433483 L18.5370192,17.6442236 L18.5380095,17.6448575 C18.7829388,17.8347189 18.8994618,18.1933125 19.0004703,18.5947683 C19.1015119,18.9965561 19.1820217,19.4339014 19.4850805,19.7457088 L19.4854105,19.7460107 L19.4857406,19.7466445 C20.0680584,20.3373581 20.2552214,20.7363991 20.2390468,20.9914593 Z" id="path5" fill="#FFFFFF"></path>
                    <path d="M8.79576133,23.1029353 L8.79545334,23.104829 L8.79545334,23.1070384 C8.7326248,23.9510222 8.26818634,24.4105736 7.55489757,24.5775397 C6.84222476,24.7445059 5.87546592,24.5782025 4.910247,24.06086 C4.90993902,24.06086 4.90963103,24.0605444 4.90932305,24.0605444 C3.84123779,23.4807395 2.57080765,23.5384991 1.75557648,23.3639579 C1.34811489,23.2768452 1.08201752,23.1455448 0.960056232,22.9018815 C0.838094939,22.6579027 0.835323091,22.2327545 1.09464483,21.5077617 L1.09587676,21.5045739 L1.09680071,21.5014176 C1.22522965,21.0958382 1.13006288,20.6520682 1.0678503,20.2354419 C1.00563773,19.8191628 0.975147402,19.440096 1.11404776,19.1762012 L1.1152797,19.1736762 C1.29298592,18.8227001 1.55353959,18.697081 1.87692181,18.5784056 C2.20061201,18.4594146 2.58405092,18.3660209 2.88679828,18.0547822 L2.88864617,18.053204 L2.89018609,18.0516259 C3.17014269,17.7489406 3.38049513,17.3692426 3.62657359,17.1000137 C3.83415418,16.872763 4.04204275,16.7222094 4.35526152,16.72 C4.35895732,16.7203472 4.36234513,16.7203472 4.36604093,16.72 C4.42086191,16.7203472 4.47907071,16.72505 4.54066733,16.734866 C4.95644446,16.7992537 5.31894053,17.097173 5.66819332,17.5826057 L6.67652987,19.4659458 L6.67683785,19.4668927 L6.67745382,19.4675239 C6.94570707,20.0416476 7.5123959,20.6732151 7.9925415,21.3174077 C8.47268709,21.9613162 8.84411467,22.6080338 8.79576133,23.1029353 Z" id="path7" fill="#FFFFFF"></path>
                    <path d="M12.2491087,7.16726091 C12.1585405,7.05037632 11.9736161,6.93916574 11.6588658,6.8540556 L11.6581771,6.85382864 L11.657144,6.85360168 C11.0025048,6.66885594 10.7184031,6.65569224 10.3530311,6.49908958 C9.75831152,6.24716356 9.26690166,6.15887598 8.85848394,6.16 C8.64463284,6.1604647 8.45350987,6.18543034 8.28236012,6.22446753 C7.78475168,6.33726683 7.45450497,6.57262474 7.24754118,6.70176519 L7.24719681,6.70199215 C7.24719681,6.70221911 7.24685245,6.70221911 7.24685245,6.70244607 C7.20621729,6.72786563 7.15387371,6.75101559 7.02714713,6.81229489 C6.89938745,6.87380115 6.70792012,6.96640099 6.43242756,7.10257721 C6.18758354,7.22354709 6.10803507,7.38105759 6.19274903,7.56557637 C6.27711863,7.75009515 6.54710134,7.96298398 7.04092176,8.14704884 L7.04161049,8.14750276 L7.04264359,8.14772972 C7.34912906,8.26643 7.55850341,8.42643706 7.79887067,8.55376183 C7.9190543,8.61731073 8.04543651,8.67405083 8.19764615,8.71694634 C8.3498558,8.75984185 8.5275485,8.78889278 8.75000874,8.79751727 C9.27206715,8.81748978 9.65637927,8.71422281 9.99557949,8.58621716 C10.3354684,8.45843847 10.6233237,8.30206278 10.9536049,8.2314781 L10.9542936,8.23125114 L10.9550168,8.23102418 C11.6320053,8.09167051 12.1148405,7.81092053 12.2659826,7.54424209 C12.341743,7.41078939 12.3393325,7.28414551 12.2491087,7.16726091 Z" id="path9" fill="#FFFFFF"></path>
                    <path d="M11.3726201,8.45896388 C10.9113746,8.61308612 10.3726148,8.8 9.79932958,8.8 C9.22633925,8.8 8.77366805,8.63023812 8.44809866,8.46482425 C8.28531397,8.38221184 8.15319885,8.29997752 8.05352272,8.24023949 C7.88058344,8.15273026 7.90129898,8.02999034 7.97235273,8.03361394 C8.09145185,8.04314518 8.10945961,8.14366955 8.18445823,8.18863036 C8.28590377,8.24950266 8.41300558,8.32833418 8.56694328,8.40659857 C8.87481868,8.5629383 9.28531922,8.71511906 9.79932958,8.71511906 C10.3124552,8.71511906 10.9114481,8.52200697 11.2770725,8.3905298 C11.4842163,8.31604176 11.7477987,8.18252234 11.9629086,8.08130191 C12.1274839,8.00386043 12.1214827,7.91060973 12.2573687,7.92076236 C12.3932544,7.930915 12.2927322,8.02397759 12.1024352,8.13044155 C11.9121378,8.2369055 11.6144296,8.37816478 11.3726201,8.45896388 L11.3726201,8.45896388 Z" id="path11" fill="#202020"></path>
                    <path d="M19.3158744,17.7296228 C19.2556185,17.7272792 19.1961956,17.7276014 19.1384388,17.7290369 C19.1331629,17.7293299 19.1278871,17.7293299 19.1223335,17.7293299 C19.2714461,17.2324754 18.9415658,16.8659867 18.0624403,16.4464729 C17.1508265,16.0234728 16.4244234,16.0653363 16.3016902,16.9236992 C16.2939152,16.9685215 16.2875286,17.0145157 16.2828081,17.0608029 C16.2144996,17.0859971 16.146191,17.1176365 16.0773271,17.1571856 C15.6494268,17.404441 15.4156227,17.8526646 15.2856699,18.4028664 C15.1559947,18.952453 15.1185083,19.6165856 15.0829656,20.363332 C15.0829656,20.363625 15.0829656,20.363625 15.0829656,20.363918 C15.0610292,20.7392247 14.9144157,21.2471821 14.7661361,21.7850798 C13.272789,22.909125 11.2002076,23.3960189 9.44029055,22.1286886 C9.3211671,21.929771 9.18427232,21.7326112 9.04349007,21.5380881 C8.95352271,21.4139037 8.86105625,21.2905397 8.76942283,21.1686991 C8.9499129,21.168992 9.10346831,21.1376164 9.22758994,21.0784684 C9.38197838,21.0043504 9.49027242,20.8859959 9.54414177,20.7336292 C9.6513251,20.4292473 9.54358641,19.9997729 9.20009992,19.5090706 C8.85661342,19.0186612 8.27488002,18.4652662 7.4201901,17.9121641 C7.4201901,17.9121641 7.4201901,17.9121641 7.4201901,17.9118711 C6.79208464,17.4996812 6.44110086,16.9945947 6.27671606,16.4462092 C6.11205357,15.8975015 6.13510077,15.3042643 6.26199918,14.7186441 C6.50524426,13.5945696 7.1300176,12.5012555 7.52876182,11.8151511 C7.63594516,11.7319514 7.56708125,11.9698322 7.12501941,12.8358119 C6.72905195,13.62741 5.98848729,15.4542581 7.00228616,16.8804002 C7.02949851,15.8655692 7.25913742,14.8305535 7.64483082,13.8623319 C8.20657147,12.5191258 9.38142302,10.1892419 9.47472251,8.33248291 C9.52303831,8.36939544 9.68825615,8.487164 9.76184056,8.53140045 C9.76211824,8.5316934 9.76211824,8.5316934 9.76239592,8.5316934 C9.9778733,8.66557457 10.139759,8.86126959 10.3494052,9.03909425 C10.5596067,9.21721187 10.8220115,9.37101409 11.2185343,9.39532949 C11.2565761,9.39767315 11.2937848,9.39884497 11.3301605,9.39884497 C11.7389011,9.39884497 12.0576743,9.2582258 12.3231336,9.09797853 C12.6116678,8.92396231 12.8421119,8.73119686 13.0606715,8.65619996 C13.0609492,8.65590701 13.0612269,8.65590701 13.0615046,8.65590701 C13.5232815,8.50356957 13.8900929,8.23404949 14.0988782,7.92 C14.457637,9.41173506 15.2917788,11.5664309 15.8280009,12.6178522 C16.1131752,13.1756416 16.680164,14.360944 16.9251029,15.7891075 C17.0803243,15.784098 17.2513734,15.8078567 17.4343626,15.8573371 C18.0749635,14.1051636 16.8912263,12.2182301 16.3497561,11.6926659 C16.1312242,11.4688471 16.1206725,11.3686559 16.2292442,11.3733432 C16.8162535,11.9214651 17.5873349,13.0232749 17.8678164,14.2671686 C17.9957978,14.8343033 18.0230379,15.4307923 17.8858377,16.0193421 C17.9527578,16.0486378 18.0210664,16.0805701 18.090208,16.1151682 C19.118446,16.6433397 19.4985859,17.1026957 19.3158744,17.7296228 Z" id="path13" fill="#FFFFFF"></path>
                    <path d="M12.3199456,5.03657795 C12.3215977,5.31792142 12.2856653,5.5574398 12.2065728,5.80197719 C12.1615541,5.94139417 12.1097412,6.05850444 12.0475824,6.16 C12.0264979,6.14633714 12.0046287,6.13323194 11.9818922,6.12068441 C11.9032127,6.07523447 11.8336196,6.03787072 11.7712748,6.00608365 C11.7088888,5.97429658 11.6602168,5.95257681 11.6100147,5.92915475 C11.6463807,5.86976312 11.7180183,5.79977579 11.7446785,5.71194309 C11.785154,5.57949696 11.8049788,5.450118 11.8086959,5.29592281 C11.8086959,5.28978847 11.8101415,5.28449062 11.8101415,5.27724094 C11.8124131,5.12945894 11.7979575,5.00314715 11.7659488,4.87376819 C11.7324945,4.73797605 11.6899332,4.64038416 11.6284146,4.55924347 C11.5666687,4.47810279 11.5051294,4.4412967 11.4311995,4.4379507 C11.4276889,4.43767186 11.4243848,4.43767186 11.4208741,4.43767186 C11.3514875,4.4379507 11.2911872,4.47029544 11.2288012,4.5405616 C11.1633382,4.6144526 11.1148088,4.70897731 11.0743539,4.84058695 C11.0340849,4.97219658 11.0142602,5.10269087 11.0103159,5.25772256 C11.009717,5.26385691 11.009717,5.26915475 11.009717,5.2752891 C11.0082714,5.36061229 11.0124016,5.4386858 11.0221075,5.51452864 C10.88003,5.41888859 10.69826,5.34915082 10.5727032,5.3087199 C10.5654754,5.23538657 10.5613452,5.15982256 10.5601062,5.08063371 L10.5601062,5.0591635 C10.5578346,4.77865653 10.5919084,4.53802281 10.6718269,4.29376426 C10.7517455,4.04922687 10.8506628,3.87356147 10.9898284,3.73051965 C11.1292212,3.58775665 11.2661566,3.52223067 11.4282446,3.52 L11.4358854,3.52 C11.5944835,3.52 11.7301591,3.58301648 11.8695519,3.71936629 C12.0110098,3.8582256 12.1130247,4.03166033 12.1952148,4.27424588 C12.2757529,4.51069708 12.3145764,4.74185044 12.3185207,5.01594423 C12.3185,5.02319392 12.3185,5.02932826 12.3199456,5.03657795 L12.3199456,5.03657795 Z" id="path15" fill="#FFFFFF"></path>
                    <path d="M9.6730722,5.53073752 C9.64422972,5.53658268 9.61624395,5.54283097 9.58854374,5.54948236 C9.43148072,5.58777829 9.30677668,5.63004207 9.18626651,5.68627661 C9.19797485,5.62742182 9.19968826,5.56776081 9.19055005,5.50104528 C9.18969334,5.49741724 9.18969334,5.49439388 9.18969334,5.49076585 C9.1771283,5.40228211 9.15057037,5.32810895 9.10630716,5.25312956 C9.05918825,5.17512682 9.00635797,5.12010162 8.93696467,5.07777455 C8.87413946,5.03947863 8.81474108,5.02174157 8.74906018,5.02214468 C8.74249209,5.02214468 8.73563844,5.02234624 8.72878478,5.02274935 C8.65510794,5.02718362 8.59399615,5.05257986 8.53602562,5.10236456 C8.47834065,5.1519477 8.44035996,5.2136243 8.41294532,5.29545664 C8.38553069,5.37708742 8.37839146,5.4573073 8.38981423,5.54941907 C8.38981423,5.55304711 8.3909565,5.55607047 8.3909565,5.55969851 C8.40352154,5.64898847 8.4289372,5.72316163 8.47405712,5.79814102 C8.52031932,5.87533754 8.57400631,5.93036273 8.64339961,5.9726898 C8.65510794,5.97974432 8.66653071,5.98619415 8.67795347,5.99183776 C8.60599005,6.03114148 8.5576392,6.0590197 8.49824082,6.08965644 C8.46026013,6.10920751 8.41514021,6.13258818 8.36259549,6.16 C8.24808227,6.08421438 8.15869914,5.98907925 8.0804532,5.86350893 C7.9879288,5.71516262 7.93852534,5.56661475 7.92367575,5.39125973 L7.92367575,5.38984883 C7.90996843,5.21449382 7.9342418,5.06372881 8.00249282,4.90772332 C8.07102941,4.75171782 8.16241153,4.63884563 8.29520118,4.54612918 C8.42770525,4.45321118 8.5613516,4.40644984 8.72241259,4.40060467 C8.73497763,4.40020156 8.7472571,4.4 8.75953657,4.4 C8.9054624,4.40020156 9.03568192,4.43446633 9.17047054,4.5104535 C9.31668194,4.59289052 9.42719719,4.69830508 9.51972158,4.84685295 C9.61253155,4.99540082 9.66193501,5.14394869 9.67564232,5.31930371 L9.67564232,5.32071461 C9.68221041,5.3942831 9.68135371,5.46361887 9.6730722,5.53073752 L9.6730722,5.53073752 Z" id="path17" fill="#FFFFFF"></path>
                    <path d="M9.68218733,6.99786317 C9.71811364,7.059618 9.90396415,7.04938234 10.011341,7.0790013 C10.1055623,7.10499066 10.181345,7.1619535 10.2872861,7.16359148 C10.388395,7.16515599 10.5457523,7.14484023 10.5589056,7.09113163 C10.5762823,7.02017443 10.3827937,6.97508256 10.2582858,6.94908725 C10.0980664,6.91563434 9.89279314,6.89866176 9.74249681,6.94341426 C9.70805568,6.95366896 9.6704659,6.97771695 9.68218733,6.99786406 L9.68218733,6.99786317 Z" id="path28396-7" fill="#202020"></path>
                    <path d="M9.67781267,6.99786317 C9.64188636,7.059618 9.45603585,7.04938234 9.34865904,7.0790013 C9.25443773,7.10499066 9.178655,7.1619535 9.07271393,7.16359148 C8.97160495,7.16515599 8.81424767,7.14484023 8.80109438,7.09113163 C8.78371775,7.02017443 8.97720629,6.97508256 9.10171425,6.94908725 C9.26193357,6.91563434 9.46720686,6.89866176 9.61750319,6.94341426 C9.65194432,6.95366896 9.6895341,6.97771695 9.67781267,6.99786406 L9.67781267,6.99786317 Z" id="path5461" fill="#202020"></path>
                </g>
            </g>
        </g>
    </g>
</svg></div>
                    <div class="subtitle">Download with this CLI</div>
                    <div class="card_option1">
                        <div class="card_copy" id="install-linux">sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && sudo chmod a+x /usr/local/bin/edgectl</div>
                        <a href="#" class="button" @click=${this.copyLinuxInstallToKeyboard.bind(this)}><?xml version="1.0" encoding="UTF-8"?>
<svg width="16px" height="15px" viewBox="0 0 16 15" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-640.000000, -241.000000)" fill="#505050" fill-rule="nonzero">
            <g id="cta-buttons" transform="translate(628.000000, 229.000000)">
                <path d="M27,16 L27,26 L16,26 L16,16 L27,16 Z M28,15 L15,15 L15,27 L28,27 L28,15 Z M12,25 L12,12 L26,12 L26,13.2380952 L13.3333333,13.2380952 L13.3333333,25 L12,25 Z" id="Shape"></path>
            </g>
        </g>
    </g>
</svg> Copy to Clipboard</a>
                    
                    </div>
                    <div class="subtitle2">Or download the binary:</div>
                    <div class="card_option2">
                        <a href="https://metriton.datawire.io/downloads/linux/edgectl" class="cta_download"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 19 20"><defs><style>.cls-1{fill:#2d8fff;}</style></defs><title>download</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><g id="Screen"><g id="login"><g id="Group-4"><g id="iconmonstr-download-19"><path id="Shape" class="cls-1" d="M9.5,17.5,2.59,10H7.77V0h3.46V10h5.18Zm7.77-.83v1.66H1.73V16.67H0V20H19V16.67Z"/></g></g></g></g></g></g></svg> Download edgectl for Linux</a>
                        <div class="subtitle3">make it executable:</div>
                        <div class="card_copy">chmod a+x ~/Downloads/edgectl</div>
                        <div class="subtitle4">and place it somewhere in your shell PATH.</div>
                    </div>
                </div>
                <div class="card">
                    <div class="title">Windows <?xml version="1.0" encoding="UTF-8"?>
<svg width="24px" height="24px" viewBox="0 0 24 24" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
    <g id="Screen" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
        <g id="login" transform="translate(-743.000000, -436.000000)" fill="#000000" fill-rule="nonzero">
            <g id="Group-4" transform="translate(69.000000, 436.000000)">
                <g id="Windows_logo_2012-Black" transform="translate(674.000000, 0.000000)">
                    <path d="M0,3.39796994 L9.80807682,2.06631845 L9.81236422,11.497787 L0.00912721311,11.5534425 L0,3.39796994 Z M9.80340466,12.5845769 L9.81101754,22.0244214 L0.00774480173,20.68075 L0.00719513518,12.5214224 L9.80340466,12.5845769 Z M10.9923526,1.89215709 L23.9969631,0 L23.9969631,11.3779578 L10.9923526,11.4808819 L10.9923526,1.89215709 Z M24,12.6733002 L23.9969494,24 L10.9923417,22.1701947 L10.9741202,12.652192 L24,12.6733002 Z" id="path13"></path>
                </g>
            </g>
        </g>
    </g>
</svg></div>
                    <div class="subtitle">Download the binary:</div>
                    <div class="card_option2">
                        <a href="https://metriton.datawire.io/downloads/windows/edgectl.exe" class="cta_download"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 19 20"><defs><style>.cls-1{fill:#2d8fff;}</style></defs><title>download</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><g id="Screen"><g id="login"><g id="Group-4"><g id="iconmonstr-download-19"><path id="Shape" class="cls-1" d="M9.5,17.5,2.59,10H7.77V0h3.46V10h5.18Zm7.77-.83v1.66H1.73V16.67H0V20H19V16.67Z"/></g></g></g></g></g></g></svg> Download edgectl for Windows</a>
                        <div class="subtitle3">and place it somewhere in your Windows System PATH.</div>
                    </div>
                </div>
            </div>
        </div>
           ${this.renderDebugDialogBox()}
    </div>    ` ;
  }

  renderDebugDialogBox() {
    if( hasDebugBackend() ) {
      return html`
      <div id="debug-dev-loop-box" style="position:absolute; background-color:red; top: 5px; left: 500px;"><div>
        <h3>Debug</h3>
        1. <button style="margin-right: 1em" @click=${this.copyLoginToKeyboard.bind(this)}>Copy edgectl command to clipboard</button>
        2. <button @click="${this.enterDebugDetails.bind(this)}">Enter the URL+JWT</button>
      </div></div>
` } else {
      return html``
    }
  }
  enterDebugDetails() {
    let the_whole_url = prompt("Enter the edgectl url w/ JST", "");
    let segments = the_whole_url.split('#');
    if( segments.length > 1 ) {
      updateCredentials(segments[1]);
    } else {
      updateCredentials(segments[0]);
    }
    window.location.reload();
  }
}

customElements.define('dw-login-page', LoginPage);

export class LoginGate extends LitElement {
  static get properties() {
    return {
      authenticated: Boolean,
      loading: Boolean,
      hasError: Boolean,
      os: String,
      namespace: String
    };
  }

  constructor() {
    super();

    this.hasError = false;
    this.loading = true;

    this.namespace = '';

    this.loadData();

    this.authenticated = useContext('auth-state', null)[0];
    registerContextChangeHandler('auth-state', this.onAuthChange.bind(this));
  }

  onAuthChange(auth) {
    if( this.authenticated !== auth ) {
      this.authenticated = auth;
      this.loading = false;
      localStorage.setItem("authenticated", auth);
      console.log("Authenticated status is " + this.authenticated);
    }
  }

  loadData() {
    ApiFetch('/edge_stack/api/config/pod-namespace')
    //fetch('http://localhost:9000/edge_stack/api/config/pod-namespace', { mode:'no-cors'})
      .then(data => data.text()).then(body => {
        this.namespace = body;
        //this.loading = false;
        this.hasError = false;
      })
      .catch((err) => {
        console.error(err);
        this.loading = false;
        this.hasError = true;
      });
  }

  renderError() {
    return html`
<dw-wholepage-error/>
    `;
  }

  renderLoading() {
    return html`
<p>Loading...</p>
    `;
  }

  updated(changedProperties) {
  }

  renderUnauthenticated() {
    return html`<dw-login-page namespace="${this.namespace}"></dw-login-page>`;
  }

  render() {
    if (this.hasError) {
      return this.renderError();
    } else if (this.loading) {
      return this.renderLoading();
    } else if (!this.authenticated) {
      return this.renderUnauthenticated();
    } else {
      return html`<slot></slot>`;
    }
  }
}

customElements.define('login-gate', LoginGate);
