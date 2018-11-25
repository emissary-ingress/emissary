import React from 'react';
import Helmet from 'react-helmet';
import { graphql } from 'gatsby';
import Layout from '../components/Layout';
import Sidebar from '../components/Sidebar';
import DocFooter from '../components/DocFooter';

export default ({ data, location }) => {
  const page = data.markdownRemark || {};
  const title = page.headings && page.headings[0] ? page.headings[0].value : 'Docs';

  return (
    <React.Fragment>
      <Helmet>
        <title>{title} | Ambassador</title>
        <meta name="og:title" content={`${title} | Ambassador`} />
        <meta name="og:type" content="article" />
      </Helmet>
      <Layout location={location}>
        <Sidebar location={location} />
        <div className="doc-body">
          <main className="main-body">
            <div dangerouslySetInnerHTML={{ __html: page.html }} />
            <DocFooter page={page} />
          </main>
        </div>
      </Layout>
    </React.Fragment>
  );
};

export const query = graphql`
  query($slug: String!) {
    markdownRemark(fields: { slug: { eq: $slug } }) {
      html
      fields {
        slug
      }
      headings(depth: h1) {
        value
      }
      parent {
        ... on File {
          relativePath
        }
      }
    }
  }
`;
