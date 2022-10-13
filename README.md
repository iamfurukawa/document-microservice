To build a new image:

docker build --tag=iamfurukawa/document-server:13 .
docker run --name document-server -p6661:6661 iamfurukawa/document-server:13

docker push iamfurukawa/document-server:13