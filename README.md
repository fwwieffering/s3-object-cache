# S3 Object Service
[![Build Status](https://travis-ci.com/fwwieffering/s3-object-cache.svg?branch=master)](https://travis-ci.com/fwwieffering/s3-object-cache)
## What is it?
S3 is an AWS's implementation of an object store. It is excellent and industry standard. However, migrating to using S3 as an object store can have some hiccups if you previously used a filesystem for distributing objects.

- While S3 writes are immediate, S3 updates are _eventually consistent_, and can take up to 24 hours to propagate. During this period applications that request the object could pull different versions of it. Because of this it is best practice to store objects in S3 at versioned paths, but that may not be compatible with some applications.
- Fetching objects from S3 is a network request and adds a lot of latency for object requests compared to pulling from a filesystem. To mitigate this applications must implement a local cache. This is an additional app update to migrate to S3.

This project attempts to solve those problems by doing the following:
- storing objects at versioned paths, but allowing for the setting of a default version of an object so unversioned fetches are supported.
- Implementing a [sidecar / daemonset](https://github.com/fwwieffering/s3-object-cache/blob/master/sidecar) to solve the caching problem at an infrastructure level. Instead of requesting the object service directly, clients can request the local daemonset which will handle the caching and allow for higher performance.

## Endpoints
- `GET` `/`: List Categories
- `GET` `/{category}`: List objects in category `{category}`
- `GET` `/{category}/{object name}/versions`: List versions for object `{object}` in category `{category}`
- `POST` `/{category}/{object name}/{version}`: Add an object with object name `{object name}` to the object service with version `{version}`. The object content must be sent in the body of the HTTP request. `{category}` provides a way of bucketing object types
  - Object versions can not be overwritten. If a POST is sent with the same object name and version an error will be returned.
  - adding an Object does not set the default object version
- `GET /{category}/{object name}/{version}`: get the object content of version `{version}` of object `{object name}`. The object content will be returned in the body.
- `GET` `/{category}/{object name}`: Get the default version of an object. The object content will be returned in the body.
  - This allows for unversioned fetches.
  - There is a default dev version as well as a default version for each object. To request the dev version, supply query param `dev=true`, e.g. `/object/{object_name}?dev=true`
  - If a specific version has not been set as default for an object, an error is returned
- `PUT` `/{category}/{object name}/{version}`: Set the default version of object `{object name}` to `{version}`. This controls the object version returned when an object is requested without a specific version at `GET /object/{object name}`
  - Adding a new object version does not automatically set the default version of an object. This must be done in a separate step
  - The "dev" version of an object can be set by providing query param `?dev=true`. This controls the default dev version of an object.
  - Setting the "prod" version (no `?dev=true` param) will also set the dev version to the same value


## Deployment
[Check out the deployment section](./deployment)

## Future Work
- add api authentication
- error code improvements, currently all errors are returned as 5XX
