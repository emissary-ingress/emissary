import { terser } from 'rollup-plugin-terser';

export default [{
  output: {
    format: "esm"
  },
  plugins: [
    terser({
      include: [/^.+\.js$/],
    })
  ]
}];
