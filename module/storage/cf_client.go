package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"org.commonjava/charon/module/util"
)

var DEFAULT_BUCKET_TO_DOMAIN = map[string]string{
	"prod-ga":         "maven.repository.redhat.com",
	"prod-maven-ga":   "maven.repository.redhat.com",
	"prod-ea":         "maven.repository.redhat.com",
	"prod-maven-ea":   "maven.repository.redhat.com",
	"stage-ga":        "maven.stage.repository.redhat.com",
	"stage-maven-ga":  "maven.stage.repository.redhat.com",
	"stage-ea":        "maven.stage.repository.redhat.com",
	"stage-maven-ea":  "maven.stage.repository.redhat.com",
	"prod-npm":        "npm.registry.redhat.com",
	"prod-npm-npmjs":  "npm.registry.redhat.com",
	"stage-npm":       "npm.stage.registry.redhat.com",
	"stage-npm-npmjs": "npm.stage.registry.redhat.com",
}

const (
	ENDPOINT_ENV                = "aws_endpoint_url"
	INVALIDATION_BATCH_DEFAULT  = 3000
	INVALIDATION_BATCH_WILDCARD = 15

	INVALIDATION_STATUS_COMPLETED  = "Completed"
	INVALIDATION_STATUS_INPROGRESS = "InProgress"
)

type CFCLient struct {
	awsProfile string
	client     *cloudfront.Client
}

func NewCFClient(awsProfile string) (*CFCLient, error) {
	cfClient := &CFCLient{
		awsProfile: awsProfile,
	}

	var cfg aws.Config
	var err error
	if !util.IsBlankString(cfClient.awsProfile) {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsProfile))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	// Create an Amazon cloudfront service client
	cfClient.client = cloudfront.NewFromConfig(cfg)

	return cfClient, nil
}
