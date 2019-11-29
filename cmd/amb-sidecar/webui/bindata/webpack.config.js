const path = require('path');

module.exports = {
  entry: {
    main: [
      './edge_stack/vendor/polyfill.js',
      './edge_stack/vendor/lit-element.min.js',
      './edge_stack/vendor/rapidoc.min.js',
      './edge_stack/components/context.js',
      './edge_stack/components/login-gate.js',
      './edge_stack/components/snapshot.js',
      './edge_stack/components/tabs.js',
      './edge_stack/components/signup.js',
      './edge_stack/components/dashboard.js',
      './edge_stack/components/add-button.js',
      './edge_stack/components/hosts.js',
      './edge_stack/components/mappings.js',
      './edge_stack/components/limits.js',
      './edge_stack/components/services.js',
      './edge_stack/components/resolvers.js',
      './edge_stack/components/debugging.js',
      './edge_stack/components/routetable.js',
      './edge_stack/components/apis.js',
      './edge_stack/components/documentation.js',
      './edge_stack/components/wholepage-error.js',
      './edge_stack/components/support.js',
      ]
  },
  output: {
    filename: '[hash]-bundle.js',
    path: path.resolve(__dirname, './edge_stack/')
  }
};
