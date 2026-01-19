package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"privatebox/internal/config"
	"privatebox/internal/providers"

	// AWS SDK v2
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"

	// Pulumi AWS
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type AWSProvider struct {
	cfg config.Profile
}

func NewAWSProvider(cfg config.Profile) *AWSProvider {
	return &AWSProvider{cfg: cfg}
}

func (p *AWSProvider) Name() string {
	return "aws"
}

func (p *AWSProvider) GetSSHUser() string {
	// For Amazon Linux 2 or Ubuntu, it varies.
	// We'll default to "ubuntu" for now as we'll use Ubuntu AMIs by default,
	// or "ec2-user" for Amazon Linux.
	// To be safe, let's assume Ubuntu for this MVP.
	return "ubuntu"
}

func (p *AWSProvider) GetPulumiProgram(spec providers.InstanceSpec) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		// 1. Create Security Group
		sg, err := ec2.NewSecurityGroup(ctx, spec.Name+"-sg", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Allow SSH"),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(22),
					ToPort:     pulumi.Int(22),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(spec.Name + "-sg"),
			},
		})
		if err != nil {
			return err
		}

		// 1.5 Create IAM Role for SSM Support
		// We create a role that allows EC2 to assume it, and attach the SSM Core policy.
		role, err := iam.NewRole(ctx, spec.Name+"-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Action": "sts:AssumeRole",
					"Principal": {
						"Service": "ec2.amazonaws.com"
					},
					"Effect": "Allow",
					"Sid": ""
				}]
			}`),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(spec.Name + "-role"),
			},
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, spec.Name+"-rpa", &iam.RolePolicyAttachmentArgs{
			Role:      role.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		})
		if err != nil {
			return err
		}

		instanceProfile, err := iam.NewInstanceProfile(ctx, spec.Name+"-profile", &iam.InstanceProfileArgs{
			Role: role.Name,
		})
		if err != nil {
			return err
		}

		// 2. Import Key Pair (if provided)
		// We assume the user has a key file locally. We need to read it or rely on an existing key name.
		// For simplicity/portability, we'll upload the public key to AWS.
		var keyName pulumi.StringInput
		if p.cfg.SSHPublicKey != "" {
			// In a real app, read the file. Here we assume the path is valid.
			// However, Pulumi's KeyPair resource needs the *content* of the public key, not the path.
			// Let's assume the user config provided the path, so we read it.
			// BUT, inside Pulumi RunFunc we are in the cloud execution context essentially.
			// It's better to pass the key content via config or read it here if it's local.

			// We will try to read the key file.
			keyContent, err := p.readPublicKey(p.cfg.SSHPublicKey)
			if err != nil {
				return fmt.Errorf("failed to read ssh key: %w", err)
			}

			key, err := ec2.NewKeyPair(ctx, spec.Name+"-key", &ec2.KeyPairArgs{
				PublicKey: pulumi.String(keyContent),
			})
			if err != nil {
				return err
			}
			keyName = key.KeyName
		}

		// 3. Find AMI (Ubuntu 22.04 LTS)
		amiID := p.cfg.AWS.AMI
		if amiID == "" {
			// Lookup latest Ubuntu 22.04
			mostRecent := true
			ubuntu, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
				MostRecent: &mostRecent,
				Filters: []ec2.GetAmiFilter{
					{
						Name:   "name",
						Values: []string{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"},
					},
					{
						Name:   "virtualization-type",
						Values: []string{"hvm"},
					},
				},
				Owners: []string{"099720109477"}, // Canonical
			})
			if err != nil {
				return err
			}
			amiID = ubuntu.Id
		}

		// 4. Create Instance
		instanceType := p.cfg.AWS.InstanceType
		if instanceType == "" {
			instanceType = "t3.micro"
		}

		// Prepare tags
		pulumiTags := pulumi.StringMap{}
		pulumiTags["Name"] = pulumi.String(spec.Name)
		if spec.UserDataName != "" {
			pulumiTags["UserDataName"] = pulumi.String(spec.UserDataName)
		}
		for k, v := range spec.Tags {
			pulumiTags[k] = pulumi.String(v)
		}

		srv, err := ec2.NewInstance(ctx, spec.Name, &ec2.InstanceArgs{
			InstanceType:        pulumi.String(instanceType),
			VpcSecurityGroupIds: pulumi.StringArray{sg.ID()},
			Ami:                 pulumi.String(amiID),
			KeyName:             keyName,
			UserData:            pulumi.String(spec.UserData),
			Tags:                pulumiTags,
			IamInstanceProfile:  instanceProfile.Name,
		})
		if err != nil {
			return err
		}

		// 5. Export Outputs
		ctx.Export("instanceID", srv.ID())
		ctx.Export("publicIP", srv.PublicIp)
		ctx.Export("publicDNS", srv.PublicDns)
		if v, ok := pulumiTags["UserDataName"]; ok {
			ctx.Export("userDataName", v)
		} else {
			ctx.Export("userDataName", pulumi.String(""))
		}
		return nil

	}
}

func (p *AWSProvider) readPublicKey(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("ssh public key path is empty")
	}

	// Handle tilde expansion
	if strings.HasPrefix(path, "~/") {
		dirname, _ := os.UserHomeDir()
		path = filepath.Join(dirname, path[2:])
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetInstanceStatus uses AWS SDK to fetch real-time info
func (p *AWSProvider) GetInstanceStatus(ctx context.Context, instanceID string) (*providers.RuntimeInfo, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(p.cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := awsec2.NewFromConfig(cfg)

	resp, err := client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	inst := resp.Reservations[0].Instances[0]

	state := string(inst.State.Name)
	ip := ""
	if inst.PublicIpAddress != nil {
		ip = *inst.PublicIpAddress
	}

	return &providers.RuntimeInfo{
		ID:       instanceID,
		PublicIP: ip,
		State:    state,
		// CPUUsage requires CloudWatch, skipping for MVP
		CPUUsage: 0.0,
	}, nil
}
