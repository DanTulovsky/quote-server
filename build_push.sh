export IMAGE_NAME="server"
export IMAGE_ID="ghcr.io/dantulovsky/quote-server/$IMAGE_NAME"
export VERSION="0.0.2"

echo "Building local/$IMAGE_NAME"
docker build . --file Dockerfile --tag local/$IMAGE_NAME

echo "Tagging local/$IMAGE_NAME $IMAGE_ID:$VERSION"
docker tag local/$IMAGE_NAME $IMAGE_ID:$VERSION
docker tag local/$IMAGE_NAME $IMAGE_ID:latest

echo "Pushing $IMAGE_ID:$VERSION"
docker push $IMAGE_ID:$VERSION
docker push $IMAGE_ID:latest
