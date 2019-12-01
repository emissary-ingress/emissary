import { terser } from 'rollup-plugin-terser';

export default [{
  output: {
    externalLiveBindings: false,
    format: "esm"
  },
  plugins: [
    terser({
      include: [/^.+\.js$/],
    })
  ],
  external: [ // Listing dependencies here will reduce the number of warnings
    '/edge_stack/vendor/lit-element.min.js',
    '/edge_stack/components/snapshot.js',
    '/edge_stack/components/resources.js',
    '/edge_stack/components/cookies.js',
    '/edge_stack/components/context.js',
    '/edge_stack/components/request-labels.js'
  ]
}];
