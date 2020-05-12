import React, { Component } from 'react';
import classnames from 'classnames';

import Navigation from './Navigation';

import styles from './styles.module.css';

class Sidebar extends Component {
  constructor() {
    super();

    this.state = {
      open: false,
      expandAll: false
    };
  }

  toggleSidebar = () => {
    this.setState({
      open: !this.state.open
    });
  };

  toggleExpandAll = () => {
    this.setState({
      expandAll: !this.state.expandAll
    });
  };

  render() {
    const { location, items, prefix } = this.props;
    const { open, expandAll } = this.state;
    return (
      <div>
        <div className={classnames(styles.Sidebar, open && styles.open,)}>
          <div className={styles.ExpandAllContainer}>
            <button
              className={styles.ExpandAllButton}
              onClick={this.toggleExpandAll}
            >
              { expandAll ? 'Collapse All' : 'Expand All' }
            </button>
          </div>
          <Navigation prefix={prefix} items={items} location={location} expandAll={expandAll} />
        </div>
        <button
          onClick={this.toggleSidebar}
          className={classnames(styles.FloatingButton, open && styles.open)}
        >
          <div className={styles.arrows}>
            <svg
              className={styles.left}
              viewBox="0 0 926.23699 573.74994"
              version="1.1"
              x="0px"
              y="0px"
              width="15"
              height="15"
            >
              <g transform="translate(904.92214,-879.1482)">
                <path
                  d="
          m -673.67664,1221.6502 -231.2455,-231.24803 55.6165,
          -55.627 c 30.5891,-30.59485 56.1806,-55.627 56.8701,-55.627 0.6894,
          0 79.8637,78.60862 175.9427,174.68583 l 174.6892,174.6858 174.6892,
          -174.6858 c 96.079,-96.07721 175.253196,-174.68583 175.942696,
          -174.68583 0.6895,0 26.281,25.03215 56.8701,
          55.627 l 55.6165,55.627 -231.245496,231.24803 c -127.185,127.1864
          -231.5279,231.248 -231.873,231.248 -0.3451,0 -104.688,
          -104.0616 -231.873,-231.248 z
        "
                  fill="currentColor"
                />
              </g>
            </svg>
            <svg
              className={styles.right}
              viewBox="0 0 926.23699 573.74994"
              version="1.1"
              x="0px"
              y="0px"
              width="15"
              height="15"
            >
              <g transform="translate(904.92214,-879.1482)">
                <path
                  d="
          m -673.67664,1221.6502 -231.2455,-231.24803 55.6165,
          -55.627 c 30.5891,-30.59485 56.1806,-55.627 56.8701,-55.627 0.6894,
          0 79.8637,78.60862 175.9427,174.68583 l 174.6892,174.6858 174.6892,
          -174.6858 c 96.079,-96.07721 175.253196,-174.68583 175.942696,
          -174.68583 0.6895,0 26.281,25.03215 56.8701,
          55.627 l 55.6165,55.627 -231.245496,231.24803 c -127.185,127.1864
          -231.5279,231.248 -231.873,231.248 -0.3451,0 -104.688,
          -104.0616 -231.873,-231.248 z
        "
                  fill="currentColor"
                />
              </g>
            </svg>
          </div>
        </button>
      </div>
    );
  }
}

export default Sidebar;
