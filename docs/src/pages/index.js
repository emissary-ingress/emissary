import React from 'react';

import Layout from '../components/Layout';
import Link from '../components/Link';

const IndexPage = ({ location }) => (
  <Layout>
    <div className="container">
      <h1>Welcome!</h1>
      <p>This is the docs dev environment for the Ambassador Api Gateway</p>
      <p><Link to="/docs">Start Here</Link></p>
    </div>
  </Layout>
);

export default IndexPage;
