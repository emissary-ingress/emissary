import { css } from '../framework/view.js'

export function controls() {
  return css`${plus()} ${minus()} ${close()} ${copy()}`
}

export function plus() {
  return css`
    .plus {
      position: relative;
      display: inline-block;
      width: 1em;
      height: 1em;
      min-width: 1em;
      min-height: 1em;
      opacity: 0.4;
    }
    .plus:hover {
      opacity: 1;
    }
    .plus:before, .plus:after {
      display: inline-block;
      position: absolute;
      left: 0.36em;
      bottom: 0;
      content: ' ';
      height: 0.7em;
      width: 0.28em;
      background-color: #000;
      border-radius: 0.14em;
    }
    .plus:before {
      transform: rotate(90deg);
    }
`
}

export function minus() {
  return css`
    .minus {
      position: relative;
      display: inline-block;
      width: 1em;
      height: 1em;
      min-width: 1em;
      min-height: 1em;
      opacity: 0.4;
    }
    .minus:hover {
      opacity: 1;
    }
    .minus:before {
      display: inline-block;
      position: absolute;
      left: 0.15em;
      bottom: 0.21em;
      content: ' ';
      height: 0.28em;
      width: 0.7em;
      background-color: #000;
      border-radius: 0.14em;
    }
`
}

export function close() {
  return css`
     .close {
       position: relative;
       display: inline-block;
       width: 22px;
       height: 22px;
       opacity: 0.4;
     }
     .close:hover {
       opacity: 1;
     }
     .close:before, .close:after {
       display: inline-block;
       position: absolute;
       top: 1px;
       left: 9px;
       content: ' ';
       height: 20px;
       width: 4px;
       background-color: #000;
       border-radius: 2px;
     }
     .close:before {
       transform: rotate(45deg);
       transform-origin: 11px, 11px;
     }
     .close:after {
       transform: rotate(-45deg);
       transform-origin: 11px, 11px;
     }
`
}

export function copy() {
  return css`
    .copy {
      position: relative;
      display: inline-block;
      width: 22px;
      height: 22px;
      opacity: 0.4;
    }
    .copy:hover {
      opacity: 1;
    }
    .copy:before, .copy:after {
      display: inline-block;
      position: absolute;
      top: 1.5px;
      left: 3px;
      content: ' ';
      height: 12px;
      width: 9px;
      background-color: #fff;
      border-radius: 2px;
      border-width: 2px;
      border-style: solid;
    }

    .copy:after {
      transform: translate(3px, 3px);
    }
`
}

export function top() {
  return css`
     .top {
       position: relative;
       display: inline-block;
       width: 22px;
       min-width: 22px;
       height: 22px;
       min-height: 22px;
       opacity: 0.4;
     }
     .top:hover {
       opacity: 1;
     }
     .top:before {
       display: inline-block;
       position: absolute;
       top: 3px;
       left: 2.5px;
       content: ' ';
       height: 4px;
       width: 17px;
       background-color: #000;
       border-radius: 2px;
     }
     .top:after {
       display: inline-block;
       position: absolute;
       top: 4px;
       left: 2.5px;
       content: ' ';
       width: 0;
       height: 0;
       border-left: 8.5px solid transparent;
       border-right: 8.5px solid transparent;
       border-bottom: 15px solid #000;
       border-radius: 15px;
     }
`
}

export function bottom() {
  return css`
     .bottom {
       position: relative;
       display: inline-block;
       width: 22px;
       min-width: 22px;
       height: 22px;
       min-height: 22px;
       opacity: 0.4;
     }
     .bottom:hover {
       opacity: 1;
     }
     .bottom:before {
       display: inline-block;
       position: absolute;
       bottom: 3px;
       left: 2.5px;
       content: ' ';
       height: 4px;
       width: 17px;
       background-color: #000;
       border-radius: 2px;
     }
     .bottom:after {
       display: inline-block;
       position: absolute;
       bottom: 4px;
       left: 2.5px;
       content: ' ';
       width: 0;
       height: 0;
       border-left: 8.5px solid transparent;
       border-right: 8.5px solid transparent;
       border-top: 15px solid #000;
       border-radius: 15px;
     }
`
}
