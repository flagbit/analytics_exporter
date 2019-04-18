FROM golang:alpine as builder

ADD . /go/src/flagbit/analytics_exporter
WORKDIR /go/src/flagbit/analytics_exporter

RUN apk --no-cache add curl git \
  && curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
  && dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dist/analytics_exporter .

FROM scratch

COPY --from=builder /go/src/flagbit/analytics_exporter/dist/analytics_exporter /analytics_exporter

ADD ca-certificates.crt /etc/ssl/certs/

ENV SCRAPE_PORT=9090 
ENV VIEW_ID='' 
ENV VIEW_METRICS='rt:activeUsers,ga:sessions' 
ENV INTERVAL=15 
ENV START_DATE='2010-01-01'

ENTRYPOINT [ "/analytics_exporter" ]
