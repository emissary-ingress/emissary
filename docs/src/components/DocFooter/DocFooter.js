import React from 'react';

import styles from './styles.module.css';

import Link from '../Link';

const DocFooter = ({ page }) => (
  <aside className={styles.DocFooter}>
    <Link
      to={`https://github.com/datawire/ambassador/tree/master/docs/${
        page ? page.parent.relativePath : ''
      }`}
    >
      Edit this page on GitHub
    </Link>
  </aside>
);

export default DocFooter;
