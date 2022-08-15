FROM golang:1.18-alpine

WORKDIR /app

COPY . ./

RUN go mod tidy
RUN go build -o /surfacemap

EXPOSE 8000

CMD [ "/salesnearme" ]
