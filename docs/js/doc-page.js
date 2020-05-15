import React from 'react';
import Helmet from 'react-helmet';
import { graphql } from 'gatsby';
import { MDXRenderer } from 'gatsby-plugin-mdx';

import Layout from '../../../../src/components/Layout';
import Sidebar from './Sidebar';
import DocFooter from '../../../../src/components/DocFooter';

import Link from '../../../../src/components/Link';

import isAesPage from './isAesPage';
import isApiGatewayPage from './isApiGatewayPage';

import template from '../../../../src/utils/template';

import docLinks from './doc-links.yml';
import versions from './versions.yml';

function removeDoubleSlashes(str) {
  return str.replace(/\/{2,}/g, '/');
}

export default ({ data, location }) => {
  const page = data.mdx || {};
  const title =
    page.headings && page.headings[0] ? page.headings[0].value : 'Docs';

  const aesPage = isAesPage(page.fields.slug);
  const apiGatewayPage = isApiGatewayPage(page.fields.slug);

  const metaDescription = page.frontmatter
    ? page.frontmatter.description
    : null;

  // docs/version/path/to/page
  const slug = data.mdx.fields.slug || location.pathname;

  // we don't need the version, which is the first element of the array
  const [, ...rest] = slug
    // remove the docs part
    .replace(/(\/docs\/)|(docs\/)/, '')
    // get the parts of the path by splitting it
    .split('/');

  // Finally, build the canonical URL: we know the initial part (docs/latest)
  // and need to append the rest of the original slug to it
  const canonicalUrl = `https://www.getambassador.io/docs/latest${removeDoubleSlashes(
    // Also, we can add a trailing slash to make sure we're pointing to the right URL
    `/${rest.join('/')}/`,
  )}`;

  return (
    <React.Fragment>
      <Helmet>
        <title>{title} | Ambassador</title>
        <meta name="og:title" content={`${title} | Ambassador`} />
        <meta name="og:type" content="article" />
        <link rel="canonical" href={canonicalUrl} />
        {metaDescription && (
          <meta name="description" content={metaDescription} />
        )}
      </Helmet>
      <Layout location={location}>
        <Sidebar location={location} prefix="" items={docLinks} />
        <div className="doc-body">
          <main className="main-body">
            <div className="doc-tags">
              {aesPage && (
                <Link className="doc-tag aes" to="/editions">
                  Ambassador Edge Stack
                </Link>
              )}
              {apiGatewayPage && (
                <Link className="doc-tag api" to="/editions/">
                  Ambassador API Gateway
                </Link>
              )}
            </div>
            <MDXRenderer slug={page.fields.slug}>
              {template(page.body, versions)}
            </MDXRenderer>
            <div>
              <h2>Questions?</h2>
              <p>
                We’re here to help. If you have questions,{' '}
                <a href="http://d6e.co/slack">join our Slack</a>,{' '}
                <a href="/contact/">contact us</a>, or{' '}
                <a href="/demo/">request a demo</a>.
              </p>
            </div>
          </main>
        </div>
        <DocFooter page={page} branch="release/v1.4" />
      </Layout>
    </React.Fragment>
  );
};

export const query = graphql`
  query($slug: String!) {
    mdx(fields: { slug: { eq: $slug } }) {
      body
      fields {
        slug
      }
      headings(depth: h1) {
        value
      }
      frontmatter {
        description
      }
      parent {
        ... on File {
          relativePath
        }
      }
    }
  }
`;
