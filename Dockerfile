FROM golang:1.22-alpine

WORKDIR /app

COPY . ./

RUN go mod tidy
RUN go build -o /surfacemap

EXPOSE 8000

CMD [ "/surfacemap" ]
