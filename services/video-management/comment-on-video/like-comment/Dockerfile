FROM golang:1.23.6 as build
WORKDIR /video-management/like-comment
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY main.go .
RUN go build -tags lambda.norpc -o main main.go
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /video-management/like-comment/main ./main
ENTRYPOINT [ "./main" ]
