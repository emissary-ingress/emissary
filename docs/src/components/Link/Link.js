import React from 'react';
import { Link as GatsbyLink } from 'gatsby';
import { Link as AnchorLink } from 'react-scroll';

const Link = ({ children, to, ...other }) => {
  // Tailor the following test to your environment.
  // This example assumes that any internal link (intended for Gatsby)
  // will start with exactly one slash, and that anything else is external.
  const internal = /^\/(?!\/)/.test(to);
  const anchor = /^#/.test(to);

  // Use Gatsby Link for internal links, and <a> for others
  if (anchor) {
    return (
      <AnchorLink
        to={to.substring(1)}
        smooth={true}
        className={other.className}
      >
        {children}
      </AnchorLink>
    );
  } else if (internal) {
    return (
      <GatsbyLink to={to} {...other}>
        {children}
      </GatsbyLink>
    );
  }
  return (
    <a
      href={to}
      target="_blank"
      rel="noopener noreferrer"
      className={other.className}
    >
      {children}
    </a>
  );
};

export default Link;
