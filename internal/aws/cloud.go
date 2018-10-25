package aws

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/acm/acmiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/aws/aws-sdk-go/service/wafregional/wafregionaliface"
)

type CloudAPI interface {
	ACMAPI
	EC2API
	EC2MetadataAPI
	ELBV2API
	IAMAPI
	ResourceGroupsTaggingAPIAPI
	WAFRegionalAPI
}

type Cloud struct {
	acm         acmiface.ACMAPI
	ec2         ec2iface.EC2API
	ec2metadata *ec2metadata.EC2Metadata
	elbv2       elbv2iface.ELBV2API
	iam         iamiface.IAMAPI
	rgt         resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	wafregional wafregionaliface.WAFRegionalAPI
	clusterName string
}

// Cloudsvc is a pointer to the Cloud service
// TODO: Deprecate global variable
var Cloudsvc CloudAPI

func NewCloudsvc(awsSession *session.Session) {
	Cloudsvc = &Cloud{
		acm.New(awsSession),
		ec2.New(awsSession),
		ec2metadata.New(awsSession),
		elbv2.New(awsSession),
		iam.New(awsSession),
		resourcegroupstaggingapi.New(awsSession),
		wafregional.New(awsSession),
		"TODO GET RID OF THIS",
	}
}