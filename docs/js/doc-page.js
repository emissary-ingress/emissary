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

function relativeToAbsUrl(slug) {
  return 'https://www.getambassador.io' + getPathFromSlug(slug);
}

function getMainDocsUrl(slug) {
  // All doc pages slugs follow the pattern /docs/{VERSION}/{PATH}
  // This regex extracts the /docs/{VERSION} part to find the current page's main docs slug
  const mainDocsSlug = getPathFromSlug(slug).match(/\/docs\/[^/]*/)
  // If, for some reason, we can't find this part (regex matches nothing in the slug string), we assume /docs/latest/
  return relativeToAbsUrl(mainDocsSlug ? mainDocsSlug[0] : '/docs/latest/')
}

// Used to get a flat array of *all* links with their corresponding parents
function flattenLinks(links, parent) {
  return links.reduce((acc, cur) => {
    let link = cur;
    if (parent) {
      link = { ...cur, parent };
    }
    if (!link.items) {
      return [...acc, link];
    }
    return [...acc, link, ...flattenLinks(link.items, link)];
  }, []);
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
}) => ({
  '@context': 'http://schema.org',
  '@type': isFAQ ? 'FAQPage' : 'WebPage',
  '@id': relativeToAbsUrl(slug),
  // Search engines normally penalize sites for duplicating content.
  // We want to communicate to the search engine "we're hosting multiple versions of the docs, please don't penalize us for having content that is duplicated in multiple versions".
  // Principally, we do that with assemblyVersion (below), but some search engines don't have assemblyVersion (the version of the software being described) figured out, so we set version (the version of the web page describing the software) too.
  version,
  breadcrumb: {
    '@type': 'BreadcrumbList',
    itemListElement: [
      // Every page (except the main docs pages) will have this as the root
      {
        '@type': 'ListItem',
        position: 1,
        item: {
          '@id': getMainDocsUrl(slug),
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
    ],
  },
  mainEntity: {
    '@type': 'APIReference',
    headline: title,
    inLanguage: 'en',
    isPartOf: 'https://www.getambassador.io/#software',
    // the currently selected docs version
    assemblyVersion: version,
    // the sidebar section / parent link, if applicable,
    articleSection: section,
    license: 'Apache-2.0',
    keywords:
      'Kubernertes,API Gateway,Edge Stack,Envoy Proxy,Kubernetes Ingress,Load Balancer,Identity Aware Proxy,Developer Portal, microservices, open source',
  },
});

function useDocSEO({ slug, title }) {
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
  // So we flatten all links (see flattenLinks above)
  const flatLinks = flattenLinks(docLinks);
  // Find the current expanded link
  const expandedLink = flatLinks.find(
    (l) => getPathFromSlug(l.link) === getPathFromSlug(slug),
  );

  // We also need a list of parent links to display in the Schema as breadcrumbs
  let breadcrumbs = [];
  // And if this page is inside a given sidebar section, we'll also include it in the Schema (as an articleSection property)
  let section;

  function parseBreadcrumbs(menuEntry) {
    if (!menuEntry) {
      return
    }
    // We'll only add a menu entry to the breadcrumbs array if it has a link
    if (menuEntry.link) {
      breadcrumbs = [
        { title: menuEntry.title, slug: menuEntry.link },
        ...breadcrumbs,
      ];
    }
    // If it has a parent, we'll have to process it as well
    if (menuEntry.parent) {
      // The section is a textual representation of where in the docs the current page is found
      // If it's already defined, it means we have more than one parent-level, so we separate each of them with a " > "
      section = section
        ? `${menuEntry.parent.title} > ${section}`
        : menuEntry.parent.title;
      // This process is recursive as, theoretically, we could have infinitely nested links in the docs sidebar
      parseBreadcrumbs(menuEntry.parent);
    }
  }
  if (getMainDocsUrl(slug) === relativeToAbsUrl(slug)) {
    // If the current slug is that of the main page for the current docs version (/docs/latest, /docs/1.4/, etc.), then we don't want to parse breadcrumbs for it.
    // If we did, we'd have duplicate breadcrumbs in the getDocPageSchema func()
  } else {
    parseBreadcrumbs(expandedLink);
  }

  // We only want the major and minor versions as the docs doesn't differentiate between fixes (1.5.0 and 1.5.4 have the same docs, for example)
  // 1.5.2 => [1, 5, 2] => [1, 5] => 1.5
  const docsVersion = versions.version.split('.').slice(0, 2).join('.');

  const schema = getDocPageSchema({
    slug,
    title,
    version: docsVersion,
    isFAQ: slug.includes('about/faq/'),
    breadcrumbs,
    section,
  });

  return {
    canonicalUrl,
    schema,
  };
}

export default ({ data, location }) => {
  const page = data.mdx || {};
  const title =
    page.headings && page.headings[0] ? page.headings[0].value : 'Docs';

  const aesPage = isAesPage(page.fields.slug);
  const apiGatewayPage = isApiGatewayPage(page.fields.slug);

  const metaDescription = page.frontmatter
    ? page.frontmatter.description
    : page.excerpt;

  // docs/version/path/to/page
  const slug = data.mdx.fields.slug || location.pathname;

  const { canonicalUrl, schema } = useDocSEO({ slug, title });

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
        <script type="application/ld+json">{JSON.stringify(schema)}</script>
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
      excerpt(pruneLength: 150, truncate: true)
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
