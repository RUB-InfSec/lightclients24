FROM node:21
COPY . /implementation
WORKDIR /implementation
RUN ln -sf python3 /usr/bin/python && \
    yarn install && \
    yarn build
