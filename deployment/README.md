# Deploying

## Resources
The s3-object-cache requires a couple resources for its backend.
- an S3 Bucket for object storage
- a DynamoDB table for setting default item versions

These items are described in CloudFormation template [resources.yml](resources/resources.yml).

## API
The s3-object-cache api is a docker container published to [fwieffering/s3-object-cache](https://hub.docker.com/r/fwieffering/s3-object-cache/)

### Docker configuration
| Environment variable | mandatory | description |
| -------------------- | --------- | ----------- |
| `S3_BUCKET`          | yes       | the name of the s3 bucket produced by [resources.yml](resources/resources.yml) |
| `DYNAMO_TABLE`       | yes       | the name of the dynamo table produced by [resources.yml](resources/resources.yml) |
| `S3_PATH_PREFIX`     | no        | the (optional) s3 path prefix to put all objects under |

### Fargate Template
A CloudFormation template for running the API in AWS Fargate is provided in [api/fargate/api.json](api/fargate/api.json). It requires some parameters to be provided, which can be viewed in the template.

example:
```bash
$ aws cloudformation create-stack --stack-name example-stack --template-body file://deployment/api/fargate/api.json --parameters ParameterKey=VpcId,ParameterValue=vpc-1234576 ParameterKey=SubnetList,ParameterValue=\"subnet-111111,subnet-22222,subnet-33333\" ParameterKey=DNSZone,ParameterValue=example.com. ParameterKey=DNSName,ParameterValue=object-cache.example.com ParameterKey=CertificateId,ParameterValue=cda9d2ed-a190-43be-8170-027d79f1d840 ParameterKey=S3Bucket,ParameterValue=example-bucket ParameterKey=DynamoDBTable,ParameterValue=example-table --capabilities CAPABILITY_IAM
```

## Sidecar

The sidecar container is published to [fwieffering/s3-object-cache-sidecar](https://hub.docker.com/r/fwieffering/s3-object-cache-sidecar/)

The sidecar needs to be run near client applications, whether that is as a sidecar, as a daemonset on the host the clients run on, or somewhere accessible via an overlay network such as kubernetes. There is no provided deployment option for the sidecar at this time.

### Docker Configuration

| Environment Variable | default  | description |
|---------------|----------|-------------|
| `CACHE_SIZE` | 1000 | number of entries to keep in the cache |
| `CACHE_EXPIRY_SECONDS` | 300 | seconds to keep maps cached |
