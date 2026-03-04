# AWS EKS Cluster Configuration
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.30"

  cluster_endpoint_public_access       = true
  cluster_endpoint_public_access_cidrs = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  cluster_endpoint_private_access      = true

  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnets
  control_plane_subnet_ids = module.vpc.private_subnets

  # OIDC Provider required for IRSA (IAM Roles for Service Accounts)
  enable_irsa = true

  # Cluster addons — EBS CSI Driver is required for gp3 StorageClass
  cluster_addons = {
    aws-ebs-csi-driver = {
      most_recent              = true
      service_account_role_arn = module.ebs_csi_irsa.iam_role_arn
    }
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
  }

  eks_managed_node_groups = {
    system = {
      min_size     = 1
      max_size     = 3
      desired_size = 2
      
      instance_types = ["t3.medium"]
      capacity_type  = "ON_DEMAND"
      
      labels = {
        role = "system"
      }
    }
    
    workloads = {
      min_size     = 2
      max_size     = 10
      desired_size = 3
      
      instance_types = ["m5.large", "m5a.large"]
      capacity_type  = "SPOT"
      
      labels = {
        role = "workload"
      }
    }
  }

  access_entries = {
    # Allow local admin access to the cluster via CLI
    admin = {
      kubernetes_groups = []
      principal_arn     = data.aws_caller_identity.current.arn

      policy_associations = {
        admin = {
          policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
          access_scope = {
            type = "cluster"
          }
        }
      }
    }
  }
}

data "aws_caller_identity" "current" {}
