package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
)

func getElbDescription(elbName string) *elb.LoadBalancerDescription {
	fmt.Println("Getting ELB description for elb ", elbName)
	svc := elb.New(session.New())
	input := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			aws.String(elbName),
		},
	}
	result, err := svc.DescribeLoadBalancers(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodeDependencyThrottleException:
				fmt.Println(elb.ErrCodeDependencyThrottleException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	if len(result.LoadBalancerDescriptions) == 0 {
		return nil
	}
	description := result.LoadBalancerDescriptions[0]

	return description
}

func getInstancesFromElbDescription(description elb.LoadBalancerDescription) []*elb.Instance {
	return description.Instances
}

func createLBListenersFromDescription(elbDescription *elb.LoadBalancerDescription) []*elb.Listener {
	var listeners []*elb.Listener
	for _, value := range elbDescription.ListenerDescriptions {
		listener := &elb.Listener{}
		listener.SetInstancePort(*value.Listener.InstancePort)
		listener.SetInstanceProtocol(*value.Listener.InstanceProtocol)
		listener.SetLoadBalancerPort(*value.Listener.LoadBalancerPort)
		listener.SetProtocol(*value.Listener.Protocol)
		if value.Listener.SSLCertificateId != nil {
			listener.SetSSLCertificateId(*value.Listener.SSLCertificateId)
		}
		listeners = append(listeners, listener)
	}
	return listeners
}

func registerInstancesToElb(loadBalancerName *string, instances []*elb.Instance) {
	fmt.Println("Registering instances to new ELB...")
	fmt.Println("Target ELB name: ", *loadBalancerName)
	fmt.Println("Instances to be attached: ", instances)
	svc := elb.New(session.New())
	input := &elb.RegisterInstancesWithLoadBalancerInput{
		Instances:        instances,
		LoadBalancerName: loadBalancerName,
	}

	_, err := svc.RegisterInstancesWithLoadBalancer(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodeInvalidEndPointException:
				fmt.Println(elb.ErrCodeInvalidEndPointException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func describeELBInstanceHealth(elbName string) *elb.DescribeInstanceHealthOutput {
	fmt.Println("Describing ELB Instance health for ", elbName)
	svc := elb.New(session.New())
	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	result, err := svc.DescribeInstanceHealth(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodeInvalidEndPointException:
				fmt.Println(elb.ErrCodeInvalidEndPointException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}

	return result
}

func createLoadBalancer(input *elb.CreateLoadBalancerInput) *elb.CreateLoadBalancerOutput {
	svc := elb.New(session.New())
	result, err := svc.CreateLoadBalancer(input)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return result
}

func deleteElb(elbName string) *elb.DeleteLoadBalancerOutput {
	fmt.Println("Deleting ELB...")
	time.Sleep(5 * time.Second)
	svc := elb.New(session.New())
	input := &elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
	}

	result, err := svc.DeleteLoadBalancer(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}

	return result
}

func describeELBTags(elbName string) *elb.TagDescription {
	svc := elb.New(session.New())
	input := &elb.DescribeTagsInput{
		LoadBalancerNames: []*string{
			aws.String(elbName),
		},
	}

	result, err := svc.DescribeTags(input)
	if err != nil {

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}
	return result.TagDescriptions[0]
}

func createLbCookieStickinessPolicy(elbName string, policyName string) {
	fmt.Println("Creating ELB Cookie Stickiness Policy...")
	svc := elb.New(session.New())
	input := &elb.CreateLBCookieStickinessPolicyInput{
		CookieExpirationPeriod: aws.Int64(1800),
		LoadBalancerName:       aws.String(elbName),
		PolicyName:             aws.String(policyName),
	}

	_, err := svc.CreateLBCookieStickinessPolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodeDuplicatePolicyNameException:
				fmt.Println(elb.ErrCodeDuplicatePolicyNameException, aerr.Error())
			case elb.ErrCodeTooManyPoliciesException:
				fmt.Println(elb.ErrCodeTooManyPoliciesException, aerr.Error())
			case elb.ErrCodeInvalidConfigurationRequestException:
				fmt.Println(elb.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func setLoadBalancerPolicesOfListener(elbName string, policyNames []string) {
	fmt.Println("Setting ELB Policies of listener...")
	svc := elb.New(session.New())
	input := &elb.SetLoadBalancerPoliciesOfListenerInput{
		LoadBalancerName: aws.String(elbName),
		LoadBalancerPort: aws.Int64(443),
		PolicyNames:      []*string{},
	}
	for _, policyName := range policyNames {
		input.PolicyNames = append(input.PolicyNames, &policyName)
	}
	fmt.Println("Attempting to add policies to ELB Listener with input: ", input)

	_, err := svc.SetLoadBalancerPoliciesOfListener(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodePolicyNotFoundException:
				fmt.Println(elb.ErrCodePolicyNotFoundException, aerr.Error())
			case elb.ErrCodeListenerNotFoundException:
				fmt.Println(elb.ErrCodeListenerNotFoundException, aerr.Error())
			case elb.ErrCodeInvalidConfigurationRequestException:
				fmt.Println(elb.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func createELBPolicy(elbName string, policyName string, policyTypeName string, policyAttributes []*elb.PolicyAttributeDescription) {
	fmt.Println("Creating ELB SSL Policy...")
	defaultSSLPolicy := "ELBSecurityPolicy-2016-08"
	// "SSLNegotiationPolicy-443"
	// "SSLNegotiationPolicyType"
	svc := elb.New(session.New())
	input := &elb.CreateLoadBalancerPolicyInput{
		LoadBalancerName: aws.String(elbName),
		PolicyName:       aws.String(policyName),
		PolicyTypeName:   aws.String(policyTypeName),
		PolicyAttributes: []*elb.PolicyAttribute{
			{
				AttributeName:  aws.String("Reference-Security-Policy"),
				AttributeValue: aws.String(defaultSSLPolicy),
			},
		},
	}
	// allAttributes := input.PolicyAttributes
	// for _, policyAttribute := range policyAttributes {
	// 	allAttributes = append(input.PolicyAttributes, &elb.PolicyAttribute{AttributeName: policyAttribute.AttributeName, AttributeValue: policyAttribute.AttributeValue})
	// }
	// input.SetPolicyAttributes(allAttributes)

	_, err := svc.CreateLoadBalancerPolicy(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodePolicyTypeNotFoundException:
				fmt.Println(elb.ErrCodePolicyTypeNotFoundException, aerr.Error())
			case elb.ErrCodeDuplicatePolicyNameException:
				fmt.Println(elb.ErrCodeDuplicatePolicyNameException, aerr.Error())
			case elb.ErrCodeTooManyPoliciesException:
				fmt.Println(elb.ErrCodeTooManyPoliciesException, aerr.Error())
			case elb.ErrCodeInvalidConfigurationRequestException:
				fmt.Println(elb.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func describeELBPolicy(elbName string, policyName string) *elb.PolicyDescription {
	fmt.Println(fmt.Sprintf("Getting Policy description %s for ELB %s.", policyName, elbName))
	svc := elb.New(session.New())
	input := &elb.DescribeLoadBalancerPoliciesInput{
		LoadBalancerName: aws.String(elbName),
		PolicyNames: []*string{
			aws.String(policyName),
		},
	}

	result, err := svc.DescribeLoadBalancerPolicies(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			case elb.ErrCodePolicyNotFoundException:
				fmt.Println(elb.ErrCodePolicyNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}
	var targetPolicy *elb.PolicyDescription
	for _, policyDescription := range result.PolicyDescriptions {
		if *policyDescription.PolicyName == policyName {
			targetPolicy = policyDescription
		}
	}
	return targetPolicy
}

func configureHealthCheck(input *elb.ConfigureHealthCheckInput) {
	fmt.Println("Configuring Health Check...")
	svc := elb.New(session.New())
	_, err := svc.ConfigureHealthCheck(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elb.ErrCodeAccessPointNotFoundException:
				fmt.Println(elb.ErrCodeAccessPointNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func waitForELBInstanceInService(elbName string) {
	fmt.Println("Waiting for `InService` ELB instances states...")
	maxTries := 40
	tries := 0
	for {
		healthOutput := describeELBInstanceHealth(elbName)
		tries++
		instancesInService := true
		for _, instanceState := range healthOutput.InstanceStates {
			instanceInService := (*instanceState.State == "InService")
			instancesInService = instanceInService && instancesInService
		}
		if instancesInService == false {
			fmt.Println("Instances not in service yet. Waiting 5s and trying again.")
			time.Sleep(5 * time.Second)

		} else {
			fmt.Println("Instances in service.")
			break
		}

		if tries == maxTries {
			fmt.Println("Reached maximum number of retries for instances to become healthy. Stopping.")
			deleteElb(elbName)
			break
		}
	}

}
