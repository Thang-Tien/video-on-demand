#!/bin/bash
set -e

# Configuration
AWS_REGION=$(aws configure get region || echo "ap-southeast-2")
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
BASE_DIR="../services"

# Display usage information
usage() {
  echo "Usage: $0 [service1 service2 ...]"
  echo "  If no services are specified, all services will be deployed."
  echo "  Otherwise, only the specified services will be deployed."
  exit 1
}

# Discover all service directories
cd $(dirname "$0")
ALL_SERVICE_DIRS=$(find ${BASE_DIR} -maxdepth 1 -mindepth 1 -type d -printf "%f\n")

# Determine which services to process
if [ $# -gt 0 ]; then
  # Validate that specified services exist
  for SERVICE in "$@"; do
    if [ ! -d "${BASE_DIR}/${SERVICE}" ]; then
      echo "Error: Service '${SERVICE}' not found in ${BASE_DIR}/"
      usage
    fi
  done
  SERVICE_DIRS="$@"
  echo "Deploying specific services: $SERVICE_DIRS"
else
  SERVICE_DIRS="$ALL_SERVICE_DIRS"
  echo "Deploying all services"
fi

# Convert SERVICE_DIRS to an array for proper counting
readarray -t SERVICE_ARRAY <<< "$SERVICE_DIRS"

# Initialize counters for progress tracking
TOTAL_COUNT=${#SERVICE_ARRAY[@]}
PROCESSED_COUNT=0

# Log in to ECR
echo "Logging in to Amazon ECR..."
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

# Process each service directory
for SERVICE in ${SERVICE_ARRAY[@]}; do
  # Construct the function name based on directory name
  FUNCTION_NAME="vod-${SERVICE}"
  SERVICE_PATH="${BASE_DIR}/${SERVICE}"
  ECR_REPOSITORY="${FUNCTION_NAME}"
  IMAGE_TAG="latest"

  # Update counters for progress tracking
  PROCESSED_COUNT=$((PROCESSED_COUNT+1))
  REMAINING_COUNT=$((TOTAL_COUNT-PROCESSED_COUNT))
  
  echo -e "\033[1;32mProgress: $PROCESSED_COUNT/$TOTAL_COUNT services processed\033[0m"
  
  if [ $REMAINING_COUNT -gt 0 ]; then
    # Get remaining services
    REMAINING_SERVICES=$(printf "%s," "${SERVICE_ARRAY[@]:$PROCESSED_COUNT}")
    REMAINING_SERVICES=${REMAINING_SERVICES%,}  # Remove trailing comma
    echo -e "\033[1;33mRemaining services ($REMAINING_COUNT): $REMAINING_SERVICES\033[0m"
  fi
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
done

echo "All specified Lambda functions have been built and deployed successfully!"