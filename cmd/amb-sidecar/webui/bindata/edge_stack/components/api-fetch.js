import {getCookie} from "./cookies.js";

let ApiRootUrl = ""; // no trailing slash as all fetched urls start with a /

var urlParams = new URLSearchParams(window.location.search);
if( urlParams.has('debug-backend') ) {
  ApiRootUrl = urlParams.get('debug-backend');
  document.cookie = "debug-backend=" + ApiRootUrl;
} else {
  let cookie = getCookie("debug-backend");
  if( cookie ) {
    ApiRootUrl = cookie;
  }
}

export function ApiFetch(url, init_values) {
  let the_url = url; // the url is sometimes a string and sometimes a URL object
  if( ApiRootUrl ) { // if we are in the special debug mode that was enabled by the ?debug-backend= in the url..
    if( !((typeof(the_url) === 'string') || (the_url instanceof String)) ) { // ..if url is a URL object then..
      the_url = '' + the_url; // ..convert it into a string so that the pattern matching below will work.
    }
    if( the_url.startsWith('http') ) { // If the url string has a domain (is not just a path), then..
      let m = the_url.match(/^[^\/]*\/\/[^\/]*(.*)$/)
      the_url = ApiRootUrl + m[1] // ..replace the domain with our special debug-backend domain.
    } else {
      the_url = ApiRootUrl + the_url // ..if the url string is just a path (no domain), then prepend our special debug-backend.
    }
  }
  return fetch(the_url, init_values );
}

