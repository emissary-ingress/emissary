import React, { Component } from 'react';
import classnames from 'classnames';

import styles from './styles.module.css';

import Link from '../Link';

const ambassadorLogo = require('../../images/ambassador-logo.svg');

const isDocLink = (pathname = null) => {
  if (pathname) {
    const root = pathname.substr(1).split('/')[0];
    return (
      root === 'docs' ||
      root === 'about' ||
      root === 'concepts' ||
      root === 'user-guide' ||
      root === 'reference'
    );
  }
  return false;
};

class Header extends Component {
  constructor() {
    super();

    this.state = {
      open: false,
    };
  }

  toggleNav = () => {
    this.setState({
      open: !this.state.open,
    });
  };

  render() {
    const { location } = this.props;
    const { open } = this.state;

    return (
      <header className={styles.Header}>
        <div className={styles.Container}>
          <div className={styles.MobileTop}>
            <Link to="/" className={styles.LogoLink}>
              <img
                src={ambassadorLogo}
                alt="Ambassador Logo"
                className={styles.LogoImage}
              />
            </Link>
            <button
              onClick={this.toggleNav}
              className={classnames(styles.Burger, open && styles.open)}
            >
              <span />
              <span />
              <span />
              <span />
            </button>
          </div>
          <nav className={classnames(styles.MobileNav, open && styles.open)}>
            <ul className={styles.NavContainer}>
              <li>
                <Link
                  className={classnames(
                    styles.NavLink,
                    isDocLink((location || {}).pathname) && styles.NavLinkActive,
                  )}
                  to="/docs"
                >
                  Docs
                </Link>
              </li>
            </ul>
          </nav>
        </div>
      </header>
    );
  }
}

export default Header;
