import React from 'react';
import Item from './Item';

const Navigation = ({ items, location, prefix, expandAll }) => {
  return (
    <ul>
      {items.map((item, index) => (
        <Item key={index} prefix={prefix} item={item} currentPage={location} expandAll={expandAll} />
      ))}
    </ul>
  );
};

export default Navigation;
