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
  ]
}];
