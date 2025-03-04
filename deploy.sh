#!/bin/bash
set -e

# Configuration
AWS_REGION=$(aws configure get region || echo "ap-southeast-2")
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
BASE_DIR="services"

# Discover all service directories
cd $(dirname "$0")
SERVICE_DIRS=$(find ${BASE_DIR} -maxdepth 1 -mindepth 1 -type d -printf "%f\n")

# Log in to ECR
echo "Logging in to Amazon ECR..."
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

# Process each service directory
for SERVICE in $SERVICE_DIRS; do
  # Construct the function name based on directory name
  FUNCTION_NAME="vod-${SERVICE}"
  SERVICE_PATH="${BASE_DIR}/${SERVICE}"
  ECR_REPOSITORY="${FUNCTION_NAME}"
  IMAGE_TAG="latest"
  
  echo "Processing Lambda function: $FUNCTION_NAME from directory $SERVICE_PATH"
  
  # Check if repository exists, create if it doesn't
  if ! aws ecr describe-repositories --repository-names $ECR_REPOSITORY --region $AWS_REGION &> /dev/null; then
    echo "Creating ECR repository: $ECR_REPOSITORY"
    aws ecr create-repository --repository-name $ECR_REPOSITORY --region $AWS_REGION
  fi
  
  # Build the Docker image
  echo "Building Docker image for $FUNCTION_NAME..."
  cd $SERVICE_PATH
  
  # Check if Dockerfile exists
  if [ ! -f "Dockerfile" ]; then
    echo "Warning: No Dockerfile found in $SERVICE_PATH, skipping..."
    cd - > /dev/null
    continue
  fi
  
  docker buildx build --platform linux/amd64 --provenance=false -t $ECR_REPOSITORY:$IMAGE_TAG .
  
  # Tag the image for ECR
  FULL_IMAGE_NAME="${ECR_REGISTRY}/${ECR_REPOSITORY}:${IMAGE_TAG}"
  echo "Tagging image as $FULL_IMAGE_NAME"
  docker tag $ECR_REPOSITORY:$IMAGE_TAG $FULL_IMAGE_NAME
  
  # Push the image to ECR
  echo "Pushing image to ECR..."
  docker push $FULL_IMAGE_NAME
  
  # Go back to the original directory
  cd - > /dev/null
  
  echo "Deployment completed for $FUNCTION_NAME"
  echo "------------------------------------"

  # Update counters for progress tracking
  PROCESSED_COUNT=$((PROCESSED_COUNT+1))
  REMAINING_COUNT=$((TOTAL_COUNT-PROCESSED_COUNT))
  
  echo "Progress: $PROCESSED_COUNT/$TOTAL_COUNT services processed"
  
  if [ $REMAINING_COUNT -gt 0 ]; then
    REMAINING_SERVICES=$(echo "$SERVICE_DIRS" | grep -v "^$SERVICE$" | sed -n "1,${REMAINING_COUNT}p" | paste -sd "," -)
    echo "Remaining services ($REMAINING_COUNT): $REMAINING_SERVICES"
  fi

done

echo "All Lambda functions have been built and deployed successfully!"