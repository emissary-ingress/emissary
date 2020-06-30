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

function getPathFromSlug(str) {
  return removeDoubleSlashes(`/${str}/`);
}

function relativeToAbsUrl(url) {
  if (url.startsWith('http') || url.startsWith('mailto')) {
    return url;
  } else
    return 'https://www.getambassador.io' + removeDoubleSlashes(`/${url}/`);
}

const getDocPageSchema = ({
  slug,
  // an array of { title, slug } relative to parent pages (only if applicable)
  // This doesn't include the root docs page nor the page itself
  breadcrumbs = [],
  // the doc page title (required)
  title,
  // the current docs version (required)
  version,
  // the current sidebar section (only if applicable)
  section,
  isFAQ,
  isLatest,
}) => ({
  '@context': 'http://schema.org',
  '@type': isFAQ ? 'FAQPage' : 'WebPage',
  '@id': relativeToAbsUrl(slug),
  // Not every search engine has assemblyVersion figured out (see below),
  // we want to separate this doc page from others of different versions 😉
  version,
  breadcrumb: {
    '@type': 'BreadcrumbList',
    itemListElement: [
      // Every doc page will have this as the root
      {
        '@type': 'ListItem',
        position: 1,
        item: {
          // be sure to update this /latest with the current version
          '@id': isLatest
            ? relativeToAbsUrl('/docs/latest/')
            : relativeToAbsUrl(`/docs/${version}/`),
          name: 'Ambassador Docs',
        },
      },
      ...breadcrumbs.map((crumb, i) => ({
        '@type': 'ListItem',
        // start at position 2
        position: i + 2,
        item: {
          '@id': relativeToAbsUrl(crumb.slug),
          name: crumb.title,
        },
      })),
      {
        '@type': 'ListItem',
        position: 1 + breadcrumbs.length + 1,
        item: {
          '@id': relativeToAbsUrl(slug),
          name: title,
        },
      },
    ],
  },
  mainEntity: {
    '@type': 'APIReference',
    headline: title,
    inLanguage: 'en',
    isPartOf: 'https://www.getambassador.io/#software',
    // the current docs version
    assemblyVersion: version,
    // the sidebar section, if applicable,
    articleSection: section,
    // @TODO: check this license
    license: 'Apache-2.0',
    // @TODO: check this programmingModel
    programmingModel: 'unmanaged',
    // @TODO: adding keywords related to this doc page would be optimal
    keywords:
      'Kubernertes,API Gateway,Edge Stack,Envoy Proxy,Kubernetes Ingress,Load Balancer,Identity Aware Proxy,Developer Portal, microservices, open source',
  },
});

// Used to get a flat array of *all* links with their corresponding parents
function flatAllLinks(links, parent) {
  return links.reduce((acc, cur) => {
    let link = cur;
    if (parent) {
      link = { ...cur, parent };
    }
    if (!link.items) {
      return [...acc, link];
    }
    return [...acc, link, ...flatAllLinks(link.items, link)];
  }, []);
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

  // For Schema purposes, we want to make the relationship between pages and their parents explicit
  // So we flat all links (see flatAllLinks above)
  const flatLinks = flatAllLinks(docLinks);
  // Find the current expanded link
  const expandedLink = flatLinks.find(
    (l) => getPathFromSlug(l.link) === getPathFromSlug(slug),
  );
  let breadcrumbs = [];
  let section;
  // And check to see if it has any parent
  if (expandedLink && expandedLink.parent) {
    if (expandedLink.parent.link) {
      // If it does, that's the Schema section and part of the breadcrumbs
      section = expandedLink.parent.title;
      breadcrumbs.push({
        title: expandedLink.parent.title,
        slug: expandedLink.parent.link,
      });
    }
    // If the parent also has a parent, then that's also part of the breadcrumbs
    if (expandedLink.parent.parent && expandedLink.parent.parent.link) {
      breadcrumbs = [
        {
          title: expandedLink.parent.parent.title,
          slug: expandedLink.parent.parent.link,
        },
        ...breadcrumbs,
      ];
    }
  }
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
        <script type="application/ld+json">
          {JSON.stringify(
            getDocPageSchema({
              isLatest: slug.includes('docs/latest'),
              slug,
              title,
              version: versions.version,
              isFAQ: slug.includes('about/faq/'),
              breadcrumbs,
              section,
            }),
          )}
        </script>
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
        <DocFooter page={page} branch="master" />
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
