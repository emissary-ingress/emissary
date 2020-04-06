import React, { Component } from 'react';
import classnames from 'classnames';

import Chevron from './Chevron';
import Link from '../../../../../src/components/Link';

import isAesPage from '../isAesPage';

import styles from './styles.module.css';

const matchPath = (path1 = '', path2 = '') => path1.replace(/\//g, '') === path2.replace(/\//g, '');

const hasActiveChild = (location, item) => {
  return (
    matchPath(item.link, location.pathname) ||
    (
      item.items &&
      item.items.find(i => {
        if (i.items) return hasActiveChild(location, i);
        // to account for matches when the url has a trailing slash
        return matchPath(i.link, location.pathname);
      })
    )
  );
};

const isActive = (location, item) => {
  const currentItemLink = item.link || '';
  const currentLocationPath = location.pathname || '';

  return currentItemLink.replace(/\//g, '') === currentLocationPath.replace(/\//g, '');
};

const sanitizeLink = (prefix, dst, basepath) => {
  if (!dst.startsWith('/')) {
    return dst;
  }
  // FFS https://github.com/gatsbyjs/gatsby/issues/6945
  //return path.relative(basepath.replace(/\/[^/]*$/, '/'), prefix + dst);
  return `${prefix}${dst}`;
};

class Item extends Component {
  constructor(props) {
    super(props);

    this.toggle = this.toggle.bind(this);

    this.state = {
      open: this.props.item.collapsable === false ||
            isActive(this.props.currentPage, this.props.item) ||
            hasActiveChild(this.props.currentPage, this.props.item)
    };
  }

  toggle(e) {
    e.preventDefault();

    this.setState({
      open: !this.state.open,
    });
  }

  componentDidUpdate(prevProps) {
    if (prevProps.expandAll !== this.props.expandAll) {
      this.setState({
        open: this.props.expandAll,
      });
    }
  }

  render() {
    const { prefix, item, currentPage, expandAll } = this.props;

    const canToggle = item.collapsable !== false;

    const isOpen = !canToggle || this.state.open;

    const AesPage = isAesPage(item.link);

    if (item.items) {
      return (
        <li
          className={classnames(
            styles.ItemWithChildren,
            isOpen && styles.ItemWithChildrenOpen,
          )}
        >
          <button onClick={canToggle ? this.toggle : null} className={styles.ItemToggleButton}>
            {
              item.link ? (
                <Link
                  className={classnames(
                    styles.ItemLink,
                    isActive(currentPage, item) && styles.ItemLinkActive
                  )}
                  to={sanitizeLink(prefix, item.link, currentPage.pathname)}
                >
                  {item.title}
                </Link>
              ) : (
                <span
                  className={classnames(
                    styles.ItemToggleButtonText,
                    isOpen && styles.ItemToggleButtonTextActive,
                  )}
                >
                  {item.title}
                </span>
              )
            }
            { canToggle &&
              <Chevron onClick={canToggle ? this.toggle : null} isExpanded={isOpen} />
            }
          </button>
          <ul
            className={classnames(
              styles.ItemChildren,
              isOpen && styles.ItemChildrenOpen,
            )}
          >
            {item.items.map((subItem, index) => (
              <Item key={index} prefix={prefix} item={subItem} currentPage={currentPage} expandAll={expandAll} />
            ))}
          </ul>
        </li>
      );
    }
    return (
      <li className={styles.ItemLi}>
        <Link
          className={classnames(styles.ItemLink, isActive(currentPage, item) && styles.ItemLinkActive)}
          to={sanitizeLink(prefix, item.link, currentPage.pathname)}
        >
          {item.title}
          { AesPage && false /*temporary disable this marker because it looks bad*/ && <span className={styles.AesPage}></span> }
        </Link>
      </li>
    );
  }
}

export default Item;
