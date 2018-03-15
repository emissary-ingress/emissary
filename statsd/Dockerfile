FROM mhart/alpine-node:9

WORKDIR /application
RUN npm install https://api.github.com/repos/etsy/statsd/tarball/8d5363c
ADD config.js .
CMD ["node", "node_modules/statsd/stats.js", "config.js"]
