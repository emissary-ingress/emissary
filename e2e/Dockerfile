FROM buildkite/puppeteer:latest
ENV  PATH="${PATH}:/node_modules/.bin"

WORKDIR /usr/src/app

COPY . .

RUN npm install

CMD [ "npm", "test" ]
