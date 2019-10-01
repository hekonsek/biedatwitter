# Biedatwitter - poor man's Twitter implementation

## Usage

This application uses Mongo as database, so we expect that you have default Mongo server installed on your local
 machine (i.e. unsecured `mongodb://localhost:27017`).

Running backend of application:

    go run biedatwitter.go
    
Creating new tweet:

    curl http://localhost:8080/tweet -X POST -d '{"text": "My #awesome tweet! #yolo"}' -u henry:secretpass
    
Reading tweets by tag:

    curl http://localhost:8080/tweet/yolo

Displaying tag trending information

    curl http://localhost:8080/admin/trending/2005/2019/yolo -u admin:admin

Example of interaction:

```
# Hey, I wanna share something on Twitter!
$ curl http://localhost:8080/tweet -X POST -d '{"text": "My #awesome tweet! #yolo"}' -u henry:secretpass
{"status":"success","tags":["awesome","yolo"]}

# Let me see timeline for #yolo tweets like mine!
$ curl http://localhost:8080/tweet/yolo
{"tweets":[{"text":"My #awesome tweet! #yolo","author":"henry","created":"2019-10-01T08:46:58.056+02:00"}]}

# Let me check of trendy #lolo hashtag is for years between 2005 and 2019...
$ curl http://localhost:8080/admin/trending/2005/2019/yolo -u admin:admin
{"count":1}
```

## Design notes

### Dependency management

For dependency management I use Go modules with vendoring enabled. I'm fan of using Go modules for new projects as these
are official dependency management tool for Go and all the tools used by community so far (like `dep`) will become deprecated.
I'm also really into using vendoring with Go modules - I believe this is the best practice to protect against dependencies
being removed from GitHub by its author.  

### Building

Project can be built using Makefile:

    $ make
    
Default make tasks perform tests and builds binary distribution of the application. I don't have string opinions regarding
build systems for Go. I prefer makefiles as many popular open source projects use them as build system.

In order to build Docker image from application, execute the following command:

    $ make docker-build
    
To push image to Docker Hub, execute the following command:   

    $ make docker-push

As base image for Docker image of our application I used the latest Fedora. I don't like minimal Docker images like
Alpine Linux because from my experience the fact that it does not contain basic Linux administration tools (telnet, dig,
nslookup, etc) makes it more difficult to properly debug containers in production environment. Fedora is upstream version
of Red Hat / CentOS Linux, so it offers the most recent version of Linux tools and libraries while keeping production-grade 
level of Red Hat / CentOS distributions. 

### REST framework

I decided to use Gin, because I believe it provides great lightweight REST abstraction on the top of Go `http` package.
It also makes HTTP basic auth easy and I happen to use it in this implementation.

### Authentication

Biedatwitter uses basic authentication instead of OAuth, GWT and similar solutions, because it is supposed to be simple.
Of course in real-life application we should not hardcode username and passwords in codebase using plain text ;) .

### API contract

For the sake of simplicity of demo I don't provide API contract. For production use cases I would use either 
OpenAPI (I recommend [Kin OpenAPI library for this purpose](https://github.com/getkin/kin-openapi)) or
Consumer Driven Contracts ([Pact](https://github.com/pact-foundation/pact-go) is my CDC library of choice).

### Database

I decided to choose MongoDB for this project because:
- schema-less, document oriented database like Mongo enables to create demo application like this really easily
- I know you use it ;)

In real life application should use indices on query-intensive fields (tags and timestamp fields in our case), but I
skipped this step because demo :) .  

### Testing

For the purpose of the demo I create pretty limited set of REST-based tests, just to demonstrate how I approach API-testing
of the application.

I'm using [testify](https://github.com/stretchr/testify) for great testing asserts. 

Normally for
testing purposes I would use [Test Containers](https://www.testcontainers.org/) - so MongoDB would be started as new
Docker container before tests start and shot down when tests ends. I'm not using Test Comtainers in this application
because I don't want to make assumptions that you have Docker installed on your local machines - instead I'm just 
making a requirement to have local Mongo server started in any way you want (local installation, VM, Docker, etc) as
long as you bind it to localhost. But yeah, Test Containers are awesome - highly recommended :) . 

I also created some unit tests for regex logic (which makes sense because it is relatively complex logic). Normally I 
write MUCH more unit and API tests (in particular tests of corner cases), but demo :) .

### Paging

We the sake of demo application we display last 100 tweets in chronological order - this is how we usually display timelines. 
Our real life API should allow to perform paging of the results. Paging in Mongo can be achieved using 
`order` + `cursor.limit` + `cursor.limit`.

## Stage 2 - deployment

### General architecture

For the purpose of the demo I propose to deploy it to AWS.

Typical approach for scaling simple REST application like this is to scale it horizontally behind load balancer 
(AWS ALB in case of AWS). To make deployment and server management easier, let's package our application into
Docker containers and deploy them using AWS ECS (Kubernetes and AWS EKS would be an overkill). I would like to get rid 
of servers management on the behalf of dynamic infrastructure provisioning, so I will be using ECS together with Fargate.
For DNS we will be using AWS Route53 as it is highly reliable, allows for geo-located queries, weighted routing and 
integrates well with AWS ALB.

Our deployment diagram could look as follows:

```
AWS Route53
 |
 |
AWS ALB <----- HTTPS via AWS ACM
 |
 |
ECS cluster
 |
 |
Fargate Docker Containers <--- Docker image from DockerHub 
```

For HTTPS support we will be using AWS ACM with automated ACM validation based on Route53 DNS Record validation. HTTPS
will be applied on the AWS ALB level.

### Environments

I propose to create two initial environments:
- staging
- production

Staging will be used to verify changes before we agree to promote them to production. For staging environment we could
deploy only single instance of our application (to cut costs), while for production we deploy 3 instances - each in a separated AWS 
availability zone, to ensure high level of availability.

In order to support multi-AZ setup on AWS, we need to create AWS VPC setup which subnets in at least 3 availability zones.
We also need to set up routing, security groups and all the other components of proper VPC setup. 

### Cloud formation templates

In `aws` directory you can find AWS Cloud Formation scripts that provisions AWS setup described above:

- `dns.yml` provisions Route53 DNS domain (keep in mind that you have to enable DNS delegation by yourself in your hosting provider)
- `staging.yml`
- `production.yml`

We create AWS VPC per environment. As Cloud Formation does not support automated ACM HTTPS certificate validation, some
time ago I created [custom Cloud Formation resource](https://github.com/hekonsek/cloudformation-certificate) which
perform this task using AWS Lambda written in Go.

Requests coming to `biedatwitter.staging.biedatwitter.com` address are routed to staging load balancer. Request
coming to `biedatwitter.production.biedatwitter.com` are routed to production (you can easily change URL to make it nicer).   

### Persistence setup

Considering fact that we're using AWS environment, I would recommend using [AWS DocumentDB](https://aws.amazon.com/documentdb/)
with MongoDB client compatibility enabled. That would make persistence scaling much easier comparing to scaling up
our very own self-hosted Mongo cluster. Also multi-AZ setup of Mongo is not trivial, while DocumentDB provides
multi AZ replication features out of the box.