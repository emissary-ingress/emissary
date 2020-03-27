import { Component } from "react";

export default class DetectOS extends Component {
  componentDidMount() {
    var os = "other";
    if (/Mac(intosh|Intel|PPC|68K)/.test(window.navigator.platform)) {
      os = 'mac';
    } else if (/Win(dows|36|64|CE)/.test(window.navigator.platform)) {
      os = 'windows';
    } else if (/Linux/.test(window.navigator.platform)) {
      os = 'linux';
    }

    document.querySelectorAll(`details.os-instructions[data-os="${os}"]`).forEach((el) => {
      el.open = true;
    })
  }

  render() {
    return null;
  }
}