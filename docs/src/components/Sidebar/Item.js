import React, { Component } from 'react';
import classnames from 'classnames';

import Chevron from './Chevron';
import Link from '../Link';

import styles from './styles.module.css';

const hasActiveChild = (location, item) => {
  return (
    item.items &&
    item.items.find(i => {
      if (i.items) return hasActiveChild(location, i);
      // to account for matches when the url has a trailing slash
      return i.link.replace(/\//g, '') === location.pathname.replace(/\//g, '');
    })
  );
};

const isActive = (location, item) => {
  const currentItemLink = item.link || '';
  const currentLocationPath = location.pathname || '';

  return currentItemLink.replace(/\//g, '') === currentLocationPath.replace(/\//g, '');
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

  toggle() {
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
    const { item, currentPage, expandAll } = this.props;

    const canToggle = item.collapsable !== false;

    const isOpen = !canToggle || this.state.open;

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
                  to={item.link}
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
            { canToggle && <Chevron isExpanded={isOpen} /> }
          </button>
          <ul
            className={classnames(
              styles.ItemChildren,
              isOpen && styles.ItemChildrenOpen,
            )}
          >
            {item.items.map((subItem, index) => (
              <Item key={index} item={subItem} currentPage={currentPage} expandAll={expandAll} />
            ))}
          </ul>
        </li>
      );
    }
    return (
      <li className={styles.ItemLi}>
        <Link
          className={classnames(styles.ItemLink, isActive(currentPage, item) && styles.ItemLinkActive)}
          to={item.link}
        >
          {item.title}
        </Link>
      </li>
    );
  }
}

export default Item;
