output "eks_cluster_name" {
    value = module.eks.cluster_id
}

output "eks_cluster_ca_certificate" {
    value = module.eks.cluster_certificate_authority_data
    sensitive = true
}

output "eks_cluster_endpoint" {
    value = module.eks.cluster_endpoint
}

output "eks_cluster_arn" {
    value = module.eks.cluster_arn
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}

# output "region" {
#     value = data.aws_region.current.name
# }
