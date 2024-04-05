package awscli

// import aws sdk v2 for ec2
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	log "github.com/sirupsen/logrus"
)

type AWS struct {
	EC2       *ec2.Client
	Route53   *route53.Client
	Instances map[string]ec2Types.Instance
}

func (a *AWS) Instance(name string) *ec2Types.Instance {
	if instance, ok := a.Instances[name]; ok {
		return &instance
	}

	return nil
}

func Instances(svc *ec2.Client) map[string]ec2Types.Instance {
	paginator := ec2.NewDescribeInstancesPaginator(svc, &ec2.DescribeInstancesInput{})
	instances := map[string]ec2Types.Instance{}

	for page := 1; paginator.HasMorePages(); page++ {
		output, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Fatalf("unable to describe instances, %v, page %d", err, page)
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						instances[*tag.Value] = instance
						log.Debugf("discovered instance %s on page %d", *tag.Value, page)
					}
				}
			}
		}
	}

	return instances
}

func NewAWS() *AWS {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create an EC2 client
	svc := ec2.NewFromConfig(cfg)

	route53svc := route53.NewFromConfig(cfg)

	return &AWS{
		EC2:       svc,
		Route53:   route53svc,
		Instances: Instances(svc),
	}
}

func (a *AWS) Route53IP(name, vpcID string, vpcRegion string) (string, error) {
	route53zones, err := a.Route53.ListHostedZonesByVPC(context.TODO(), &route53.ListHostedZonesByVPCInput{
		VPCId:     aws.String(vpcID),
		VPCRegion: types.VPCRegion(vpcRegion),
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to list hosted zones")
	}

	log.Debugf("Found %d zones", len(route53zones.HostedZoneSummaries))

	for _, zone := range route53zones.HostedZoneSummaries {
		zoneID := *zone.HostedZoneId
		zoneName := strings.TrimSuffix(*zone.Name, ".")

		if !strings.HasSuffix(name, zoneName) {
			continue
		}

		log.Infof("Checking zone %s/%s", zoneName, zoneID)

		records, err := a.Route53.ListResourceRecordSets(context.TODO(), &route53.ListResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneID),
		})
		if err != nil {
			return "", errors.Wrap(err, "unable to list resource record sets")
		}

		log.Infof("Found %d records", len(records.ResourceRecordSets))

		for _, record := range records.ResourceRecordSets {
			log.Infof("Found record %s", *record.Name)
			rName := strings.TrimSuffix(*record.Name, ".")
			if rName != name {
				continue
			}

			prettyJSON, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				panic(err)
			}
			log.Infof("Found record %s", string(prettyJSON))
			return *record.ResourceRecords[0].Value, nil
		}

	}

	return "", fmt.Errorf("Record %s not found", name)
}

func (a *AWS) InstanceID(name string) (string, error) {
	if instance, ok := a.Instances[name]; ok {
		return *instance.InstanceId, nil
	}

	return "", fmt.Errorf("Instance %s not found", name)
}

func (a *AWS) InstanceName(id string) string {
	for name, instance := range a.Instances {
		if *instance.InstanceId == id {
			return name
		}
	}

	return ""
}

func (a *AWS) NetworkInsights(sourceID, destID string, destIP string, port int32) (*ec2Types.NetworkInsightsPath, error) {
	nis, _ := a.EC2.DescribeNetworkInsightsPaths(context.TODO(), &ec2.DescribeNetworkInsightsPathsInput{})

	for _, ni := range nis.NetworkInsightsPaths {
		if ni.Source != nil && ni.Destination != nil && *ni.Source == sourceID && *ni.Destination == destID {
			log.Debugf("Found existing network insights path %s", *ni.NetworkInsightsPathId)
			return &ni, nil
		}
	}

	input := &ec2.CreateNetworkInsightsPathInput{
		Source:   aws.String(sourceID),
		Protocol: ec2Types.ProtocolTcp,
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeNetworkInsightsPath,
				Tags: []ec2Types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(fmt.Sprintf("%s -> %s:%d", a.InstanceName(sourceID), a.InstanceName(destID), port)),
					},
				},
			},
		},
	}

	if destIP != "" {
		input.FilterAtSource = &ec2Types.PathRequestFilter{
			DestinationAddress: aws.String(destIP),
			DestinationPortRange: &ec2Types.RequestFilterPortRange{
				FromPort: aws.Int32(port),
				ToPort:   aws.Int32(port),
			},
		}
	} else if destID != "" {
		input.Destination = aws.String(destID)
		input.DestinationPort = aws.Int32(port)
	} else {
		return nil, fmt.Errorf("Destination ID or IP must be specified")
	}

	ni, err := a.EC2.CreateNetworkInsightsPath(context.TODO(), input)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create network insights path")
	}

	return ni.NetworkInsightsPath, nil
}

func (a *AWS) RunNetworkAnalysis(ni *ec2Types.NetworkInsightsPath) (*ec2Types.NetworkInsightsAnalysis, error) {
	analysis, err := a.EC2.StartNetworkInsightsAnalysis(context.TODO(), &ec2.StartNetworkInsightsAnalysisInput{
		NetworkInsightsPathId: ni.NetworkInsightsPathId,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to start network insights analysis")
	}

	log.Infof("Network Insights Analysis ID: %s", *analysis.NetworkInsightsAnalysis.NetworkInsightsAnalysisId)

	for {
		analysis, err := a.EC2.DescribeNetworkInsightsAnalyses(context.TODO(), &ec2.DescribeNetworkInsightsAnalysesInput{
			NetworkInsightsAnalysisIds: []string{*analysis.NetworkInsightsAnalysis.NetworkInsightsAnalysisId},
		})
		if err != nil {
			return nil, errors.Wrap(err, "unable to describe network insights analysis")
		}

		log.Infof("Network Insights Analysis Status: %s", analysis.NetworkInsightsAnalyses[0].Status)

		if analysis.NetworkInsightsAnalyses[0].Status == ec2Types.AnalysisStatusSucceeded {
			return &analysis.NetworkInsightsAnalyses[0], nil
		}

		if analysis.NetworkInsightsAnalyses[0].Status == ec2Types.AnalysisStatusFailed {
			return nil, errors.New("network insights analysis failed")
		}

		time.Sleep(5 * time.Second)
	}

	return analysis.NetworkInsightsAnalysis, nil
}

func AnalysisResult(analysis *ec2Types.NetworkInsightsAnalysis) string {
	prettyJSON, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		panic(err)
	}

	ret := ""
	for _, line := range strings.Split(string(prettyJSON), "\n") {
		if strings.Contains(line, "null,") {
			continue
		}
		ret += line + "\n"
	}

	return strings.TrimSpace(ret)
}

func (a *AWS) Reachable(sourceID, destID, destIP string, port int32) (bool, error) {
	ni, err := a.NetworkInsights(sourceID, destID, destIP, port)
	if err != nil {
		return false, errors.Wrap(err, "unable to create network insights path")
	}

	log.Infof("Network Insights Path ID: %s", *ni.NetworkInsightsPathId)

	analysis, err := a.RunNetworkAnalysis(ni)
	if err != nil {
		return false, errors.Wrap(err, "unable to run network insights analysis")
	}

	if analysis == nil {
		return false, nil
	}

	log.Infof("Analysis Result: %s", AnalysisResult(analysis))

	return *analysis.NetworkPathFound, nil
}
