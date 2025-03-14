# video-on-demand

## Overview
This is a Video-on-Demand (VOD) serverless service built on AWS that handles media transcoding, processing, and delivery.

## Project Structure
```
/video-on-demand
├── deployment          # Deployment scripts and configs
│   └── deploy.sh       # Main deployment script
├── services            # Lambda functions as microservices
│   ├── custom-resource  # Custom CloudFormation resources
│   ├── dynamo          # DynamoDB integration service
│   └── ...             # Other services
└── test                # Test scripts and configuration
    └── test.sh         # Test runner script
```

## Getting Started

### Prerequisites
- AWS CLI configured with appropriate permissions
- Docker installed for local builds
- Go 1.x for development

### Deployment
Use the deployment script to build Docker images and update Lambda functions:

```bash
# Deploy all services
./deployment/deploy.sh

# Deploy specific services
./deployment/deploy.sh custom-resource dynamo

# Only build images (no Lambda updates)
./deployment/deploy.sh --build-only

# Only update Lambda functions
./deployment/deploy.sh --update-only
```

### Testing
Run tests using the test script:

```bash
# Test all services
./test/test.sh

# Test a specific service
./test/test.sh dynamo

# Run specific tests
./test/test.sh -t TestUserCreate

# Run specific test for a service
./test/test.sh dynamo -t TestUpdateItem
```
## AWS Services Used
- Lambda - Serverless compute
- DynamoDB - NoSQL database
- MediaConvert - Video transcoding
- S3 - Object storage
- CloudFront - Content delivery network
- MediaPackage - Video packaging and origination

## Trigger Mechanism
This project is triggered by adding a video to an S3 bucket. When a video is uploaded to the specified S3 bucket, an S3 event is generated, which triggers the Lambda function to start the video processing workflow.