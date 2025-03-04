FROM golang:1.23.6 as build
WORKDIR /custom-resource
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
# Copy all .go files
COPY *.go ./
# Copy presets and templates folders
COPY presets/ ./presets/
COPY templates/ ./templates/
RUN go build -tags lambda.norpc -o main .
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /custom-resource/main ./main
ENTRYPOINT [ "./main" ]
