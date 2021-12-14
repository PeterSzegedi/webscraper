FROM alpine:latest AS build
RUN apk update
RUN apk upgrade
RUN apk add --update go gcc g++ git
RUN mkdir /app 
ADD webscraper.go /app/ 
RUN go get github.com/PuerkitoBio/goquery
RUN go get github.com/sirupsen/logrus
RUN go get golang.org/x/time/rate
WORKDIR /app 

RUN CGO_ENABLED=1 GOOS=linux go build -o main

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/main .
RUN chmod +x /app/main
ARG URL=""
ARG TIMEOUT=5
ARG MAXRATE=5
ENV URL=$URL
ENV TIMEOUT=$TIMEOUT
ENV MAXRATE=$MAXRATE

ENTRYPOINT /app/main --url "$URL" --timeout $TIMEOUT --maxrate $MAXRATE
