# Video-on-Demand Serverless Processing Platform

## Overview
This is a comprehensive Video-on-Demand (VOD) serverless service built on AWS that provides end-to-end media processing, including ingestion, transcoding, metadata extraction, content protection, and multi-format delivery. The platform automatically processes uploaded videos to deliver adaptive bitrate streaming content optimized for any device or bandwidth.

## Architecture

### System Components
![VOD Architecture Diagram](docs/images/architecture.png)

The system follows a event-driven microservices architecture with the following components:

1. **Content Ingestion Layer**
   - S3 Input Bucket for raw video uploads
   - Event-driven triggers for workflow initiation
   - Validation service for media format checking

2. **Processing Layer**
   - Transcoding service powered by MediaConvert
   - Metadata extraction for content cataloging
   - Thumbnail generation at configurable intervals

3. **Storage Layer**
   - S3 for processed content, thumbnails, and metadata
   - DynamoDB for content metadata and state management

4. **Delivery Layer**
   - CloudFront CDN for global content delivery
   - MediaPackage for adaptive bitrate packaging (HLS, DASH, MSS)
   - Token-based access control for content protection

5. **Management Layer**
   - RESTful API for content management
   - Monitoring and alerting for workflow failures

## Detailed Workflow

1. **Content Upload**
   - Content creators upload raw video files to the designated S3 bucket
   - The system validates video format, resolution, and codec compatibility
   - An S3 event notification triggers the workflow Lambda function

2. **Content Processing**
   - The workflow orchestrator Lambda analyzes video attributes and determines optimal processing parameters
   - Jobs are submitted to MediaConvert with appropriate encoding profiles (4K, 1080p, 720p, 480p, etc.)
   - Parallel processing paths extract metadata, generate thumbnails, and create preview clips
   - DynamoDB tracks processing state throughout the workflow

3. **Content Packaging**
   - Transcoded assets are packaged for multi-format delivery (HLS, DASH, MSS)
   - Content protection is applied (AES-128 encryption or DRM)
   - Manifests and segment files are organized in the delivery bucket

4. **Content Delivery**
   - CloudFront distribution serves content with edge caching
   - Origin access identities secure access to S3 content
   - Signed URLs/Cookies provide time-limited viewer access

5. **Analytics and Reporting**
   - Viewer metrics are captured and processed
   - Content consumption patterns are analyzed
   - Cost allocation is tracked per content item

## Project Structure
```
video-on-demand
├─ README.md
├─ deployment
│  └─ deploy.sh                   # Main deployment script
├─ services
│  ├─ archive-source              # Permanent storage of source videos
│  ├─ auth                        # Authentication services
│  │  └─ callback                 # OAuth callback handler
│  ├─ custom-resource             # CloudFormation custom resources
│  │  ├─ presets                  # Encoding presets (various formats)
│  │  └─ templates                # Job templates for transcoding
│  ├─ dynamo                      # DynamoDB operations handler
│  ├─ encode                      # Video encoding orchestrator
│  ├─ input-validate              # Input video validation
│  ├─ media-package-assets        # AWS MediaPackage integration
│  ├─ mediainfo                   # Media metadata extraction
│  ├─ output-validate             # Processed output validation
│  ├─ profile                     # User profile services
│  ├─ profiler                    # Content profiling for optimal encoding
│  ├─ sns-notification            # Notification service
│  ├─ sqs-publish                 # Message queue publisher
│  ├─ step-functions              # Workflow orchestration
│  ├─ user-management             # User account operations
│  │  ├─ add-to-group             # User group management
│  │  └─ get-profile              # Retrieve user profiles
│  ├─ video-management            # Video content operations
│  │  ├─ comment-on-video         # Video commenting system
│  │  ├─ count-view               # View tracking and analytics
│  │  ├─ delete-video             # Video removal service
│  │  ├─ get-all-video            # Video library retrieval
│  │  ├─ get-one-video            # Individual video details
│  │  ├─ get-pre-signed-url       # Secure upload URL generation
│  │  ├─ get-trending-video       # Popularity-based recommendations
│  │  └─ update-information       # Video metadata updates
│  └─ watch-together-management   # Co-viewing experience services
│     ├─ chat-handler             # Real-time chat functionality
│     ├─ create-room              # Viewing room creation
│     ├─ websocket-connect        # WebSocket connection handler
│     └─ websocket-disconnect     # Connection termination handler
├─ test                           # Testing framework
│  ├─ lint.sh                     # Code quality checks
│  └─ test.sh                     # Test runner
└─ video-on-demand-on-aws.template  # CloudFormation template
```

## Technology Stack

### AWS Services
- **Lambda** - Serverless compute for all processing functions
- **DynamoDB** - NoSQL database for metadata and state management
- **MediaConvert** - Professional-grade video transcoding
- **S3** - Object storage for source and processed media
- **CloudFront** - Global CDN for efficient content delivery
- **MediaPackage** - Video packaging and origination for multiple formats
- **API Gateway** - RESTful API management
- **CloudWatch** - Monitoring, logging, and alerting
- **Step Functions** - Workflow orchestration for complex processing
- **IAM** - Fine-grained security controls
- **EventBridge** - Event routing between components

### Development Stack
- **Go** - Primary programming language for Lambda functions
- **Docker** - Containerization for consistent builds and testing
- **CloudFormation** - Infrastructure as Code
- **GitHub Actions** - CI/CD pipeline
- **Go Testing** - Unit and integration testing

## Getting Started

### Prerequisites
- AWS CLI configured with appropriate permissions
- Docker installed for local builds
- Go 1.x for development
- AWS account with necessary service limits

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

# Run integration tests
./test/test.sh --integration
```

## Operational Workflows

### Video Processing Workflow
1. Video uploaded to the input S3 bucket triggers an S3 event
2. The ingest-validator Lambda validates the file format and metadata
3. Upon validation, the media-processor Lambda initiates the workflow
4. Parallel processing paths begin:
   - MediaConvert job for transcoding to multiple bitrates
   - Metadata extraction for content cataloging
   - Thumbnail generation at predefined intervals
5. Upon completion, processed files are organized in the delivery bucket
6. The deliver-notifier Lambda updates DynamoDB and sends notifications
7. Content is immediately available through CloudFront CDN

### Content Management API

The platform provides a RESTful API for managing videos, rooms, chat, users, comments, and likes.

#### Video Management
- `GET /video` — Retrieve all videos  
- `GET /video/{pk}` — Get details of a specific video  
- `PUT /video/{pk}` — Update a video's information  
- `DELETE /video/{pk}` — Delete a specific video  
- `GET /presigned-url` — Generate a presigned URL for uploading a video  
- `GET /trending` — Get a list of trending videos  

#### Room Management
- `POST /rooms` — Create a new room  
- `GET /rooms/{roomId}` — Get information about a specific room  
- `GET /rooms/{roomId}/state` — Get current state of a room  

#### Chat
- `POST /chat` — Send a message in a room  

#### User
- `GET /user` — Retrieve user information  

#### Comments
- `POST /comment` — Add a comment to a video  
- `GET /comment/{videoId}` — Get comments for a specific video  
- `DELETE /comment/{commentId}` — Delete a specific comment  

#### Likes
- `POST /like/{videoId}` — Like a video  
- `POST /like/{commentId}` — Like a comment  

