package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func findHostedZone(dnsName string) *route53.HostedZone {
	fmt.Println("Finding hosted zone " + dnsName)
	svc := route53.New(session.New())
	targetDNSName := dnsName
	input := &route53.ListHostedZonesByNameInput{}
	input.SetDNSName(targetDNSName)
	result, err := svc.ListHostedZonesByName(input)
	if err != nil {
		fmt.Println(err.Error())
	}

	targetHostedZone := &route53.HostedZone{}
	for _, value := range result.HostedZones {
		if *value.Name == targetDNSName {
			fmt.Println("Found Target Hosted Zone: ", *value)
			targetHostedZone = value
		}
	}
	if targetHostedZone.Id == nil {
		fmt.Println("hosted zone not found")
		return nil
	}

	return targetHostedZone
}

func findResourceRecord(targetRecordSetName string, hostedZone *route53.HostedZone, token *string) *route53.ResourceRecordSet {
	fmt.Print("Finding resource record using name and hosted zone: ", targetRecordSetName, *hostedZone.Name)
	if token != nil {
		fmt.Println("Using token: ", *token)
	}

	svc := route53.New(session.New())

	recordSetInput := &route53.ListResourceRecordSetsInput{}
	recordSetInput.SetHostedZoneId(*hostedZone.Id)
	if token != nil {
		recordSetInput.SetStartRecordName(*token)
	} else {
		recordSetInput.SetStartRecordName(targetRecordSetName)
	}

	recordSetInput.SetMaxItems("100")
	recordSetInput.SetStartRecordType("CNAME")
	listRes, err := svc.ListResourceRecordSets(recordSetInput)
	if err != nil {
		fmt.Println(err.Error())
	}
	var recordSet *route53.ResourceRecordSet
	for _, value := range listRes.ResourceRecordSets {
		if *value.Name == targetRecordSetName {
			recordSet = value
			break
		}
	}
	if recordSet == nil && listRes.NextRecordName != nil {
		return findResourceRecord(targetRecordSetName, hostedZone, listRes.NextRecordName)
	}
	if recordSet != nil {
		return recordSet
	}

	return nil
}

func cnameBatchChange(changes []*route53.Change, hostedZone route53.HostedZone) *route53.ChangeResourceRecordSetsOutput {
	fmt.Println("cname batch change...")
	svc := route53.New(session.New())
	changeSetInput := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{},
		HostedZoneId: hostedZone.Id,
	}
	changeSetInput.ChangeBatch.SetChanges(changes)
	result, err := svc.ChangeResourceRecordSets(changeSetInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case route53.ErrCodeNoSuchHostedZone:
				fmt.Println(route53.ErrCodeNoSuchHostedZone, aerr.Error())
			case route53.ErrCodeNoSuchHealthCheck:
				fmt.Println(route53.ErrCodeNoSuchHealthCheck, aerr.Error())
			case route53.ErrCodeInvalidChangeBatch:
				fmt.Println(route53.ErrCodeInvalidChangeBatch, aerr.Error())
			case route53.ErrCodeInvalidInput:
				fmt.Println(route53.ErrCodeInvalidInput, aerr.Error())
			case route53.ErrCodePriorRequestNotComplete:
				fmt.Println(route53.ErrCodePriorRequestNotComplete, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	return result
}

func changeResourceRecordSet(action string, resourceRecordSet *route53.ResourceRecordSet, hostedZone route53.HostedZone) *route53.GetChangeOutput {
	fmt.Println("changeResourceRecordSet: ", action, *resourceRecordSet.Name, *hostedZone.Name)
	svc := route53.New(session.New())
	newResourceRecordSet := resourceRecordSet
	changeSetInput := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{},
		HostedZoneId: hostedZone.Id,
	}
	change := &route53.Change{}
	change.SetAction(action)
	change.SetResourceRecordSet(newResourceRecordSet)
	changeSetInput.ChangeBatch.SetChanges([]*route53.Change{
		change,
	})
	fmt.Println("ChangeResourceRecordSetsInput: ", changeSetInput)
	result, err := svc.ChangeResourceRecordSets(changeSetInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case route53.ErrCodeNoSuchHostedZone:
				fmt.Println(route53.ErrCodeNoSuchHostedZone, aerr.Error())
			case route53.ErrCodeNoSuchHealthCheck:
				fmt.Println(route53.ErrCodeNoSuchHealthCheck, aerr.Error())
			case route53.ErrCodeInvalidChangeBatch:
				fmt.Println(route53.ErrCodeInvalidChangeBatch, aerr.Error())
			case route53.ErrCodeInvalidInput:
				fmt.Println(route53.ErrCodeInvalidInput, aerr.Error())
			case route53.ErrCodePriorRequestNotComplete:
				fmt.Println(route53.ErrCodePriorRequestNotComplete, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	status := result.ChangeInfo.Status
	checkInterval := 5
	var changeStatusResult route53.GetChangeOutput
	for *status == "PENDING" {
		getChangeInput := &route53.GetChangeInput{
			Id: result.ChangeInfo.Id,
		}
		changeStatus, err := svc.GetChange(getChangeInput)
		changeStatusResult = *changeStatus
		if err != nil {
			fmt.Println("Error getting change info. Returning last change info result.")
			return &changeStatusResult
		}
		status = changeStatusResult.ChangeInfo.Status
		fmt.Println("Change status: " + *status)
		time.Sleep(time.Duration(checkInterval) * time.Second)
	}

	return &changeStatusResult
}

func findElbNameFromDNSRecordSet(hostedZoneDNS string, targetDNS string) string {
	fmt.Println("Determining ELB name from DNS...")
	hostedZone := findHostedZone(hostedZoneDNS)
	var sourceResourceRecord *route53.ResourceRecordSet
	if hostedZone != nil {
		sourceResourceRecord = findResourceRecord(targetDNS, hostedZone, nil)
	}
	if sourceResourceRecord == nil {
		fmt.Println("No Route53 ResourceRecord found for ", targetDNS, "Exiting.")
		os.Exit(0)
	}
	fmt.Println("Found Resource Record: ", sourceResourceRecord)
	re := regexp.MustCompile(`(internal-)(.*)(-(\d.*)\.(\w{2}\-(.*)-\d)\.elb.amazonaws.com)`)

	captureGroups := re.FindStringSubmatch(*sourceResourceRecord.ResourceRecords[0].Value)
	fmt.Println("Found ELB Name: ", captureGroups[2])
	elbName := captureGroups[2]

	return elbName
}

func deleteRecordSet(dnsName string, hostedZone *route53.HostedZone, recordSet *route53.ResourceRecordSet) {
	svc := route53.New(session.New())
	changeBatchInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZone.Id,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("DELETE"),
					ResourceRecordSet: recordSet,
				},
			},
		},
	}

	response, err := svc.ChangeResourceRecordSets(changeBatchInput)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(response)
}

func createResourceRecordSet(name string, value string, setID string) *route53.ResourceRecordSet {

	newResourceRecordSet := &route53.ResourceRecordSet{}
	newResourceRecordSet.SetName(name)
	newResourceRecordSet.SetTTL(60)
	newResourceRecordSet.SetWeight(0)
	newResourceRecordSet.SetType("CNAME")
	newResourceRecordSet.SetSetIdentifier(setID)

	newResourceRecord := &route53.ResourceRecord{}
	newResourceRecord.SetValue(value)

	newResourceRecordSet.SetResourceRecords([]*route53.ResourceRecord{
		newResourceRecord,
	})

	return newResourceRecordSet
}

func weightedBlueGreen(blueResourceRecordSet *route53.ResourceRecordSet, greenResourceRecordSet *route53.ResourceRecordSet, zone *route53.HostedZone) {
	interval := 5
	bleedAmount := int64(20)
	greenWeight := int64(0)
	blueWeight := *blueResourceRecordSet.Weight

	for greenWeight < 100 {
		blueWeight = clamp(*blueResourceRecordSet.Weight-bleedAmount, 0, 100)
		greenWeight = clamp(*greenResourceRecordSet.Weight+bleedAmount, 0, 100)
		fmt.Println("blue weight: ", blueWeight)
		fmt.Println("green weight: ", greenWeight)

		blueResourceRecordSet.SetWeight(blueWeight)
		greenResourceRecordSet.SetWeight(greenWeight)

		blueChange := &route53.Change{}
		blueChange.SetAction("UPSERT")
		blueChange.SetResourceRecordSet(blueResourceRecordSet)

		greenChange := &route53.Change{}
		greenChange.SetAction("UPSERT")
		greenChange.SetResourceRecordSet(greenResourceRecordSet)

		changes := []*route53.Change{
			blueChange,
			greenChange,
		}

		batchChangeOutput := cnameBatchChange(changes, *zone)
		if batchChangeOutput == nil {
			fmt.Println("Batch change failed. Stopping blue/green.")
			break
		}
		fmt.Println(fmt.Sprintf("Waiting %ds for next bleed of amount of %d%%.", interval, bleedAmount))
		time.Sleep(time.Duration(interval) * time.Second)
		if blueWeight == 0 && greenWeight == 100 {
			break
		}
	}
}
