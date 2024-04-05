package reach

import (
	"github.com/mhristof/go-aws-reachability/awscli"
	logger "github.com/sirupsen/logrus"
)

type Target struct {
	InstanceID string
	VPCID      string
	VPCRegion  string
	AWS        *awscli.AWS
}

func NewTarget(name string) *Target {
	a := awscli.NewAWS()
	ret := &Target{
		AWS: a,
	}

	instanceID, err := a.InstanceID(name)
	if err != nil {
		logger.Fatalf("Instance not found: %s", name)
	}

	ret.InstanceID = instanceID
	instance := a.Instance(name)
	ret.VPCID = *instance.VpcId
	ret.VPCRegion = (*instance.Placement.AvailabilityZone)[:len(*instance.Placement.AvailabilityZone)-1]
	logger.Debugf("New target instance %s/%s/%s", name, ret.VPCID, ret.VPCRegion)

	return ret
}

func (t *Target) CanReach(name string, port int32) bool {
	targetID, err := t.AWS.InstanceID(name)
	if err == nil {
		logger.Infof("Target instance %s", targetID)
		ret, err := t.AWS.Reachable(t.InstanceID, name, "", port)
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		logger.Debugf("From %s to %s:%d  = %v", t.InstanceID, name, port, ret)
		return ret
	}

	logger.Debugf("Checking if its a route53 record: %s", name)
	route53IP, err := t.AWS.Route53IP(name, t.VPCID, t.VPCRegion)
	if err == nil {
		logger.Infof("Target route53 %s", route53IP)
		ret, err := t.AWS.Reachable(t.InstanceID, name, route53IP, port)
		if err != nil {
			logger.Fatalf("Error: %s", err)
		}

		logger.Debugf("From %s to %s:%d  = %v", t.InstanceID, route53IP, port, ret)
		return ret
	}

	logger.Fatalf("Target not found: %s, %s", name, err)

	return true
}
