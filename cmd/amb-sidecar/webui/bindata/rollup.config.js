import { terser } from 'rollup-plugin-terser';

export default [{
  output: {
    externalLiveBindings: false,
    format: "esm",
    plugins: [
      terser({
        include: [/^.+\.js$/],
      })
    ],
  },
  external: id => true // Every import will be treated as an external dependency and rollup won't generate a bundle.
}];
