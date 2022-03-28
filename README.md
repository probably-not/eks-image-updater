# eks-image-updater

Sometimes, you don't need a full fledged CI/CD operator with all the bells and whistles.
Sometimes, you just need to update the image that your service is running.

This is a very simple cronjob that runs within your Kubernetes cluster, and updates your deployments to the tags that are marked by the tag you want to watch.

## Example

Let's say you have a service in the default namespace called service-a.

Let's also say that you tag each production docker image with the tag "prod".

The cronjob can be run with these flags:
```shell
./eks-image-updater --services=default/service-a --tag=prod
```

This will find the service-a Dockerfile, look for the prod tag, and update the service-a deployment image to the image tagged with the prod tag.