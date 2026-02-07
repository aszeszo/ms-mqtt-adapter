const path = require("path");
const { HtmlRspackPlugin } = require("@rspack/core");

/** @type {import('@rspack/core').Configuration} */
module.exports = {
  entry: "./src/index.ts",
  output: {
    path: path.resolve(__dirname, "../dist"),
    filename: "bundle.js",
    publicPath: "",
    clean: true,
  },
  resolve: {
    extensions: [".ts", ".js"],
  },
  module: {
    rules: [
      {
        test: /\.ts$/,
        use: {
          loader: "builtin:swc-loader",
          options: {
            jsc: {
              parser: {
                syntax: "typescript",
                decorators: true,
              },
              transform: {
                legacyDecorator: true,
                decoratorMetadata: false,
                useDefineForClassFields: false,
              },
            },
          },
        },
        type: "javascript/auto",
      },
    ],
  },
  plugins: [
    new HtmlRspackPlugin({
      template: "./src/index.html",
    }),
  ],
  devServer: {
    port: 8099,
    proxy: [
      {
        context: ["/api", "/ws"],
        target: "http://localhost:8099",
        ws: true,
      },
    ],
  },
};
