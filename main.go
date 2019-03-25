package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
)

var config = map[string]map[string]map[string][]string{
	"some-environment": {
		"us-west-2": {
			"vpc-23456789": []string{"sg-12345678", "sg-34567890"},
		},
		"us-east-1": {
			"vpc-12345678": []string{"sg-12345677", "sg-22334455", "sg-33224411"},
		},
	},
}

func replicateElb(envConfig map[string]string, sourceElbName string, newElbName string) *elb.CreateLoadBalancerInput {
	fmt.Println("Replicating ELB: ", sourceElbName)
	region := envConfig["region"]
	env := envConfig["environment"]

	sourceELBDescription := getElbDescription(sourceElbName)

	elbInput := &elb.CreateLoadBalancerInput{}
	elbName := newElbName

	elbInput.SetLoadBalancerName(elbName)
	elbInput.SetListeners(createLBListenersFromDescription(sourceELBDescription))

	elbInput.SetScheme("internet-facing")
	newSecurityGroups := []*string{}
	for _, sg := range config[env][region][*sourceELBDescription.VPCId] {
		newSecurityGroups = append(newSecurityGroups, aws.String(sg))
	}

	elbInput.SetSecurityGroups(newSecurityGroups)
	elbInput.SetSubnets(sourceELBDescription.Subnets)

	tags := describeELBTags(*sourceELBDescription.LoadBalancerName)
	elbInput.SetTags(tags.Tags)

	output := createLoadBalancer(elbInput)
	fmt.Println("ELB create output: ", output)

	// Post elb creation configuration steps

	// Set health check
	healthCheckInput := &elb.ConfigureHealthCheckInput{}
	healthCheckInput.SetHealthCheck(sourceELBDescription.HealthCheck)
	healthCheckInput.SetLoadBalancerName(elbName)
	configureHealthCheck(healthCheckInput)

	// Attach Policies
	LBCookieStickinessPolices := sourceELBDescription.Policies.LBCookieStickinessPolicies
	for _, cookiePolicy := range LBCookieStickinessPolices {
		createLbCookieStickinessPolicy(elbName, *cookiePolicy.PolicyName)
	}
	policyDescription := describeELBPolicy(sourceElbName, "some-elb-policy-name")
	createELBPolicy(elbName, "SSLNegotiationPolicy-443", "SSLNegotiationPolicyType", policyDescription.PolicyAttributeDescriptions)
	setLoadBalancerPolicesOfListener(elbName, []string{"SSLNegotiationPolicy-443"})

	// Attach instances
	instances := getInstancesFromElbDescription(*sourceELBDescription)
	registerInstancesToElb(&elbName, instances)

	waitForELBInstanceInService(elbName)
	return elbInput
}

func main() {

	// These values need to be configured prior to running.
	envConfig := map[string]string{
		"environment": "some-env",
		"region":      "us-west-2",
		"zoneValue":   "test.example.com.",
		"cnameValue":  "some-app.test.example.com",
	}

	zone := findHostedZone(envConfig["zoneValue"])
	if zone == nil {
		os.Exit(1)
	}
	cname := envConfig["cnameValue"]

	elbName := findElbNameFromDNSRecordSet(*zone.Name, cname)

	elbReplicaName := elbName + "-r"
	fmt.Println("Found ELB " + elbName)
	replConfirmation("Proceed with ELB replication? ")

	replicateElb(envConfig, elbName, elbReplicaName)
	description := getElbDescription(elbReplicaName)

	// Determine Blue Resource Record Set
	blueResourceRecordSet := findResourceRecord(cname, zone, nil)
	fmt.Println("Found blue resource record set: ", blueResourceRecordSet)

	// Create green record set whose target is the newly created ELB
	greenResourceRecordSet := createResourceRecordSet(cname, *description.DNSName, *blueResourceRecordSet.SetIdentifier+"-r")
	greenCreateChangeOutput := changeResourceRecordSet("CREATE", greenResourceRecordSet, *zone)
	fmt.Println(greenCreateChangeOutput)

	// Perform Blue/Green release
	replConfirmation("Proceed with blue/green? ")
	weightedBlueGreen(blueResourceRecordSet, greenResourceRecordSet, zone)

	// Delete Original ELB after release
	replConfirmation("Proceed with deletion of ELB " + elbName + "?")
	deleteElb(elbName)

	fmt.Println(blueResourceRecordSet)
	replConfirmation("Proceed with deletion of preceding recordset?")
	deleteRecordSet(cname, zone, blueResourceRecordSet)

	// replConfirmation("Proceed with replication back to blue configuration?")
	// Replicate ELB the internet-facing scheme to match original Name
	// replicatedElbInput.SetLoadBalancerName(elbName)
	// replicateElb(elbReplicaName, elbName)

	// Perform Blue/Green release back to original ELB with new scheme and security Groups
	// replConfirmation("Proceed with blue/green?")
	// weightedBlueGreen(greenResourceRecordSet, blueResourceRecordSet, zone)

	// Delete Replicated ELB
	// replConfirmation("Proceed with Green ELB deletion?")
	// deleteElb(*replicatedElbInput.LoadBalancerName)
}
