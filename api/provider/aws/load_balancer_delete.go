package aws

import (
	"log"
	"strings"
	"time"

	"github.com/quintilesims/layer0/common/errors"
	"github.com/quintilesims/layer0/common/retry"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elb"
	alb "github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
)

// Delete is used to delete an Elastic Load Balancer using the specified loadBalancerID.
// The associated IAM Role, IAM Role inline policy, and Security Group are also
// removed as part of the process by making DeleteRole, DeleteRolePolicy and
// DeleteSecurityGroup requests to AWS, respectively. The Load Balancer is deleted
// by making a DeleteLoadBalancer request to AWS.
func (l *LoadBalancerProvider) Delete(loadBalancerID string) error {
	fqLoadBalancerID := addLayer0Prefix(l.Config.Instance(), loadBalancerID)

	if err := l.deleteLoadBalancer(fqLoadBalancerID); err != nil {
		return err
	}

	// Check for eventually consistency
	waitUntilLBDeletedFN := func() (shouldRetry bool, err error) {
		loadBalancerName := fqLoadBalancerID
		if _, err = describeLoadBalancer(l.AWS.ELB, l.AWS.ALB, loadBalancerName); err != nil {
			if err, ok := err.(*errors.ServerError); ok && err.Code == errors.LoadBalancerDoesNotExist {
				return false, nil
			}

			return false, err
		}

		log.Printf("[DEBUG] Load Balancer not deleted, will retry lookup")
		return true, nil
	}

	if err := retry.Retry(waitUntilLBDeletedFN, retry.WithTimeout(time.Second*30), retry.WithDelay(time.Second)); err != nil {
		return errors.New(errors.EventualConsistencyError, err)
	}

	targetGroupName := fqLoadBalancerID
	if err := l.deleteTargetGroup(targetGroupName); err != nil {
		return err
	}

	roleName := getLoadBalancerRoleName(fqLoadBalancerID)
	policyName := roleName
	if err := l.deleteRolePolicy(roleName, policyName); err != nil {
		return err
	}

	if err := l.deleteRole(roleName); err != nil {
		return err
	}

	securityGroupName := getLoadBalancerSGName(fqLoadBalancerID)
	securityGroup, err := readSG(l.AWS.EC2, securityGroupName)
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return err
	}

	if securityGroup != nil {
		groupID := aws.StringValue(securityGroup.GroupId)
		if err := deleteSGWithRetry(l.AWS.EC2, groupID); err != nil {
			return err
		}
	}

	fn := waitUntilSGDeletedFN(l.AWS.EC2, securityGroupName)
	if err := retry.Retry(fn, retry.WithTimeout(time.Second*30), retry.WithDelay(time.Second)); err != nil {
		return errors.New(errors.EventualConsistencyError, err)
	}

	return l.deleteTags(loadBalancerID)
}

func (l *LoadBalancerProvider) deleteLoadBalancer(loadBalancerName string) error {
	lb, err := describeLoadBalancer(l.AWS.ELB, l.AWS.ALB, loadBalancerName)
	if err != nil {
		if err, ok := err.(*errors.ServerError); ok && err.Code == errors.LoadBalancerDoesNotExist {
			return nil
		}

		return err
	}

	if lb.isCLB {
		input := &elb.DeleteLoadBalancerInput{}
		input.SetLoadBalancerName(loadBalancerName)

		if err := input.Validate(); err != nil {
			return err
		}

		if _, err := l.AWS.ELB.DeleteLoadBalancer(input); err != nil {
			if err, ok := err.(*errors.ServerError); ok && err.Code == errors.LoadBalancerDoesNotExist {
				return nil
			}

			return err
		}
	}

	if lb.isALB {
		input := &alb.DeleteLoadBalancerInput{}
		input.SetLoadBalancerArn(aws.StringValue(lb.ALB.LoadBalancerArn))

		if err := input.Validate(); err != nil {
			return err
		}

		if _, err := l.AWS.ALB.DeleteLoadBalancer(input); err != nil {
			if err, ok := err.(*errors.ServerError); ok && err.Code == errors.LoadBalancerDoesNotExist {
				return nil
			}

			return err
		}

		waitInput := &alb.DescribeLoadBalancersInput{}
		waitInput.SetLoadBalancerArns([]*string{lb.ALB.LoadBalancerArn})

		if err := waitInput.Validate(); err != nil {
			return err
		}

		if err := l.AWS.ALB.WaitUntilLoadBalancersDeleted(waitInput); err != nil {
			return err
		}
	}

	return nil
}

func (l *LoadBalancerProvider) deleteTargetGroup(targetGroupName string) error {
	tg, err := readTargetGroup(l.AWS.ALB, aws.String(targetGroupName), nil)
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == alb.ErrCodeTargetGroupNotFoundException {
			return nil
		}

		return err
	}

	input := &alb.DeleteTargetGroupInput{
		TargetGroupArn: tg.TargetGroupArn,
	}

	if err := input.Validate(); err != nil {
		return err
	}

	retryDeleteFN := func() (shouldRetry bool, err error) {
		if _, err := l.AWS.ALB.DeleteTargetGroup(input); err != nil {
			log.Printf("[DEBUG] target group '%s' could not be deleted due to '%s', will retry.", targetGroupName, err.Error())
			if err, ok := err.(awserr.Error); ok && err.Code() == alb.ErrCodeResourceInUseException {
				return true, nil
			}

			return false, err
		}

		return false, nil
	}

	if err := retry.Retry(retryDeleteFN,
		retry.WithTimeout(time.Second*30),
		retry.WithDelay(time.Second),
	); err != nil {
		return errors.New(errors.EventualConsistencyError, err)
	}

	return nil
}

func (l *LoadBalancerProvider) deleteRolePolicy(roleName, policyName string) error {
	input := &iam.DeleteRolePolicyInput{}
	input.SetRoleName(roleName)
	input.SetPolicyName(policyName)

	if err := input.Validate(); err != nil {
		return err
	}

	if _, err := l.AWS.IAM.DeleteRolePolicy(input); err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NoSuchEntity" {
			return nil
		}

		return err
	}

	return nil
}

func (l *LoadBalancerProvider) deleteRole(roleName string) error {
	input := &iam.DeleteRoleInput{}
	input.SetRoleName(roleName)

	// todo: validate NoSuchEntity is correct error code
	if _, err := l.AWS.IAM.DeleteRole(input); err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NoSuchEntity" {
			return nil
		}

		return err
	}

	return nil
}

func (l *LoadBalancerProvider) deleteTags(loadBalancerID string) error {
	tags, err := l.TagStore.SelectByTypeAndID("load_balancer", loadBalancerID)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		if err := l.TagStore.Delete(tag.EntityType, tag.EntityID, tag.Key); err != nil {
			return err
		}
	}

	return nil
}
