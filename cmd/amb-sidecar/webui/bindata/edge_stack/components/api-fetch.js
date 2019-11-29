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
  let the_url = url;
  if( ApiRootUrl ) {
    if( !((typeof(the_url) === 'string') || (the_url instanceof String)) ) {
      the_url = '' + the_url;
    }
    if( the_url.startsWith('http') ) {
      let m = the_url.match(/^[^\/]*\/\/[^\/]*(.*)$/)
      the_url = ApiRootUrl + m[1]
    } else {
      the_url = ApiRootUrl + the_url
    }
  }
  return fetch(the_url, init_values );
}

