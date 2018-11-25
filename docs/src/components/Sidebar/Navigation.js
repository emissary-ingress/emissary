import React from 'react';
import Item from './Item';

const Navigation = ({ items, location, expandAll }) => {
  return (
    <ul>
      {items.map((item, index) => (
        <Item key={index} item={item} currentPage={location} expandAll={expandAll} />
      ))}
    </ul>
  );
};

export default Navigation;
