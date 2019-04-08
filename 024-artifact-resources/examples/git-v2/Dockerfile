FROM alpine

RUN apk add git ruby ruby-json ruby-bundler build-base ruby-dev cmake openssl-dev

ENV GEM_HOME="/usr/local/bundle"
ENV PATH $GEM_HOME/bin:$GEM_HOME/gems/bin:$PATH
ADD Gemfile .
RUN bundle install

ADD git-resource.rb /usr/bin/git-resource
RUN chmod +x /usr/bin/git-resource

WORKDIR /tmp/request

CMD ["git-resource", "info"]