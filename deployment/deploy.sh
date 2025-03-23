#!/bin/bash
set -e

# Configuration
AWS_REGION=$(aws configure get region || echo "ap-southeast-2")
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
STACK_NAME="video-on-demand-test-deploy-22"
BASE_DIR="../services"

# Display usage information
usage() {
  echo "Usage: $0 [--build-only|--update-only] [service1 service1/subservice ...]"
  echo "  --build-only     Only build and push Docker images"
  echo "  --update-only    Only update Lambda functions"
  echo "  If neither flag is provided, both operations will be performed"
  echo "  If no services are specified, all services (including nested) will be processed"
  echo "  Otherwise, only the specified services will be processed"
  exit 1
}

# Parse command line arguments
BUILD_IMAGES=true
UPDATE_LAMBDAS=true
SERVICES=()

while [[ $# -gt 0 ]]; do
  case $1 in
    --build-only)
      BUILD_IMAGES=true
      UPDATE_LAMBDAS=false
      shift
      ;;
    --update-only)
      BUILD_IMAGES=false
      UPDATE_LAMBDAS=true
      shift
      ;;
    --help)
      usage
      ;;
    *)
      SERVICES+=("$1")
      shift
      ;;
  esac
done

# Discover all service directories (including nested ones)
cd $(dirname "$0")
ALL_SERVICE_PATHS=()

# Function to find all service directories with Dockerfiles
find_service_dirs() {
  while IFS= read -r dir; do
    # Get relative path from BASE_DIR
    rel_path=${dir#$BASE_DIR/}
    if [[ -f "$dir/Dockerfile" ]]; then
      ALL_SERVICE_PATHS+=("$rel_path")
    fi
  done < <(find "$BASE_DIR" -type f -name "Dockerfile" -exec dirname {} \;)
}

find_service_dirs

# Determine which services to process
if [ ${#SERVICES[@]} -gt 0 ]; then
  # Validate that specified services exist
  SERVICE_PATHS=()
  for SERVICE in "${SERVICES[@]}"; do
    SERVICE_PATH="${BASE_DIR}/${SERVICE}"
    if [ ! -d "$SERVICE_PATH" ]; then
      echo "Error: Service '${SERVICE}' not found in ${BASE_DIR}/"
      usage
    fi
    if [ ! -f "${SERVICE_PATH}/Dockerfile" ]; then
      echo "Error: No Dockerfile found for '${SERVICE}'"
      usage
    fi
    SERVICE_PATHS+=("$SERVICE")
  done
  echo "Processing specific services: ${SERVICES[*]}"
else
  SERVICE_PATHS=("${ALL_SERVICE_PATHS[@]}")
  echo "Processing all services"
fi

# Initialize counters for progress tracking
TOTAL_COUNT=${#SERVICE_PATHS[@]}
PROCESSED_COUNT=0

# Log in to ECR if we're building images
if [ "$BUILD_IMAGES" = true ]; then
  echo "Logging in to Amazon ECR..."
  aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY
fi

# Process each service directory
for SERVICE_PATH in "${SERVICE_PATHS[@]}"; do
  # Convert service path to function name (replace / with -)
  SERVICE_NAME=$(echo "$SERVICE_PATH" | tr '/' '-')
  FUNCTION_NAME="vod-${SERVICE_NAME}"
  FULL_SERVICE_PATH="${BASE_DIR}/${SERVICE_PATH}"
  ECR_REPOSITORY="${FUNCTION_NAME}"
  IMAGE_TAG="latest"
  FULL_IMAGE_NAME="${ECR_REGISTRY}/${ECR_REPOSITORY}:${IMAGE_TAG}"

  # Update counters for progress tracking
  PROCESSED_COUNT=$((PROCESSED_COUNT+1))
  REMAINING_COUNT=$((TOTAL_COUNT-PROCESSED_COUNT))
  
  echo -e "\033[1;32mProgress: $PROCESSED_COUNT/$TOTAL_COUNT services processed\033[0m"
  
  if [ $REMAINING_COUNT -gt 0 ]; then
    # Get remaining services
    REMAINING_SERVICES=$(printf "%s," "${SERVICE_PATHS[@]:$PROCESSED_COUNT}")
    REMAINING_SERVICES=${REMAINING_SERVICES%,}  # Remove trailing comma
    echo -e "\033[1;33mRemaining services ($REMAINING_COUNT): $REMAINING_SERVICES\033[0m"
  fi
  
  echo "Processing service: $SERVICE_PATH"
  
  # Build and push Docker images if --build-only or no flag
  if [ "$BUILD_IMAGES" = true ]; then
    echo "Building Docker image for $FUNCTION_NAME..."
    cd "$FULL_SERVICE_PATH"
    
    # Check if repository exists, create if it doesn't
    if ! aws ecr describe-repositories --repository-names $ECR_REPOSITORY --region $AWS_REGION &> /dev/null; then
      echo "Creating ECR repository: $ECR_REPOSITORY"
      aws ecr create-repository --repository-name $ECR_REPOSITORY --region $AWS_REGION
    fi
    
    # Build the Docker image
    docker buildx build --platform linux/amd64 --provenance=false -t $ECR_REPOSITORY:$IMAGE_TAG .
    
    # Tag the image for ECR
    echo "Tagging image as $FULL_IMAGE_NAME"
    docker tag $ECR_REPOSITORY:$IMAGE_TAG $FULL_IMAGE_NAME
    
    # Push the image to ECR
    echo "Pushing image to ECR..."
    docker push $FULL_IMAGE_NAME
    
    # Get the current time
    BUILD_TIME=$(date "+%Y-%m-%d %H:%M:%S")
    echo "Image build completed for $FUNCTION_NAME at $BUILD_TIME"
    
    # Go back to the original directory
    cd - > /dev/null
  fi
  
  # Update Lambda functions if --update-only or no flag
  if [ "$UPDATE_LAMBDAS" = true ]; then
    LAMBDA_FUNCTION_NAME="${STACK_NAME}-${SERVICE_NAME}"
    echo "Updating Lambda function: $LAMBDA_FUNCTION_NAME..."
    
    # Check if Lambda function exists
    if ! aws lambda get-function --function-name $LAMBDA_FUNCTION_NAME --region $AWS_REGION &> /dev/null; then
      echo "Warning: Lambda function $LAMBDA_FUNCTION_NAME doesn't exist, skipping update..."
      continue
    fi
    
    # Update the Lambda function
    aws lambda update-function-code \
      --function-name $LAMBDA_FUNCTION_NAME \
      --image-uri $FULL_IMAGE_NAME \
      --publish
    
    # Get the current time
    UPDATE_TIME=$(date "+%Y-%m-%d %H:%M:%S")
    echo "Lambda update completed for $LAMBDA_FUNCTION_NAME at $UPDATE_TIME"
  fi
  
  echo "------------------------------------"
done

# Final message
if [ "$BUILD_IMAGES" = true ] && [ "$UPDATE_LAMBDAS" = true ]; then
  echo "All specified services have been built and Lambda functions updated successfully!"
elif [ "$BUILD_IMAGES" = true ]; then
  echo "All specified Docker images have been built and pushed successfully!"
elif [ "$UPDATE_LAMBDAS" = true ]; then
  echo "All specified Lambda functions have been updated successfully!"
fi
