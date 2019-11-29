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
  return fetch(ApiRootUrl + url, init_values );
}

